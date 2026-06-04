package actors

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"ergo.services/ergo/meta"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/swaggo/swag/v2"

	_ "github.com/open-source-cloud/fuse/docs" // Import generated docs
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/metrics"
)

// MuxServerFactory is a factory for creating MuxServer actors
type MuxServerFactory ActorFactory[*muxServer]

// muxServer is a mux server actor
type muxServer struct {
	act.Actor
	workers       *Workers
	config        *config.Config
	fuseMetrics   *metrics.FuseMetrics
	ergoCollector *metrics.ErgoNodeCollector
}

// NewMuxServerFactory creates a new MuxServerFactory
func NewMuxServerFactory(workers *Workers, config *config.Config, fuseMetrics *metrics.FuseMetrics) *MuxServerFactory {
	return &MuxServerFactory{
		Factory: func() gen.ProcessBehavior {
			return &muxServer{
				workers:     workers,
				config:      config,
				fuseMetrics: fuseMetrics,
			}
		},
	}
}

// Init initializes the mux server
func (m *muxServer) Init(_ ...any) error {
	m.Log().Info("starting mux server")

	// Register the Ergo node collector now that the node is available.
	m.ergoCollector = metrics.NewErgoNodeCollector(m.Node())
	m.ergoCollector.RegisterWith(m.fuseMetrics.Registry())

	// Build a combined registry that includes default Go runtime metrics.
	combinedRegistry := prometheus.NewRegistry()
	combinedRegistry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	metricsHandler := promhttp.HandlerFor(
		prometheus.Gatherers{m.fuseMetrics.Registry(), combinedRegistry},
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	)

	muxRouter := mux.NewRouter()

	// /metrics — Prometheus scrape endpoint
	muxRouter.Handle("/metrics", metricsHandler).Methods(http.MethodGet)

	// create routes
	for _, worker := range m.workers.GetAll() {
		if err := m.createWorkerPool(worker, muxRouter); err != nil {
			m.Log().Error("unable to create route for %s: %s", worker.Name, err)
			return err
		}
	}

	// When FUSE is hosted under a reverse-proxy path prefix (e.g. /fuse), the /docs redirect and
	// the Swagger UI spec URL must carry that prefix or the browser drops it. The prefix comes from
	// the proxy's X-Forwarded-Prefix header when set, else SERVER_BASE_PATH.
	configuredBasePath := strings.TrimRight(m.config.Server.BasePath, "/")
	// Serve OpenAPI doc.json from the swag/v2 registry and Swagger UI. The build-time spec carries a
	// fixed @host (localhost:9090); rewrite host/schemes/basePath from the incoming request so the
	// UI's "Try it out" targets whatever environment actually serves the docs (local, behind a
	// reverse proxy, or a production domain) without rebuilding the spec.
	muxRouter.HandleFunc("/docs/doc.json", func(w http.ResponseWriter, r *http.Request) {
		spec := swag.GetSwagger("swagger")
		if spec == nil {
			http.Error(w, "swagger spec not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(patchSwaggerServer(spec.ReadDoc(), r, configuredBasePath)))
	})
	muxRouter.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		prefix := strings.TrimRight(r.Header.Get("X-Forwarded-Prefix"), "/")
		if prefix == "" {
			prefix = configuredBasePath
		}
		http.Redirect(w, r, prefix+"/docs/index.html", http.StatusMovedPermanently)
	})
	// The UI loads its spec relatively ("doc.json" resolves under <prefix>/docs/) so it works
	// behind any prefix; a configured base path makes it explicit/absolute.
	specURL := "doc.json"
	if configuredBasePath != "" {
		specURL = configuredBasePath + "/docs/doc.json"
	}
	muxRouter.PathPrefix("/docs/").Handler(httpSwagger.Handler(httpSwagger.URL(specURL)))
	m.Log().Info("swagger documentation available at /docs or /docs/index.html")

	// create and spawn a web server meta-process
	// nolint:gosec // port is validated by the config
	port, err := strconv.Atoi(m.config.Server.Port)
	if err != nil {
		m.Log().Error("unable to convert port to int: %s", err)
		return err
	}

	// nolint:gosec // port is validated by the config
	serverOptions := meta.WebServerOptions{
		Port:        uint16(port),
		Host:        m.config.Server.Host,
		Handler:     muxRouter,
		CertManager: m.Node().CertManager(),
	}

	webserver, err := meta.CreateWebServer(serverOptions)
	if err != nil {
		m.Log().Error("unable to create Web server meta-process: %s", err)
		panic(err)
	}

	webServerID, err := m.SpawnMeta(webserver, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn Web server meta-process: %s", err)
		panic(err)
	}

	httpProtocol := "http"
	m.Log().Info("started web server %s: use %s://%s:%d/", webServerID, httpProtocol, serverOptions.Host, serverOptions.Port)
	m.Log().Info("you may check it with command below:")
	m.Log().Info("$ curl -k %s://%s:%d/health", httpProtocol, serverOptions.Host, serverOptions.Port)

	return nil
}

// createWorkerPool creates a worker pool for a given route
func (m *muxServer) createWorkerPool(webWorker WebWorker, mux *mux.Router) error {
	workerPool := meta.CreateWebHandler(meta.WebHandlerOptions{
		Worker:         webWorker.PoolConfig.Name,
		RequestTimeout: webWorker.Timeout,
	})

	workerPoolID, err := m.SpawnMeta(workerPool, gen.MetaOptions{})
	if err != nil {
		m.Log().Error("unable to spawn WebHandler meta-process: %s", err)
		return err
	}

	m.Log().Info("started worker pool %s to serve %s (meta-process: %s)", webWorker.PoolConfig.Name, webWorker.Pattern, workerPoolID)

	mux.Handle(webWorker.Pattern, workerPool)

	return nil
}

// patchSwaggerServer rewrites the host, schemes, and basePath of a Swagger 2.0 spec from the
// incoming request so the served doc.json reflects the environment actually serving it rather
// than the build-time @host. It honors the standard reverse-proxy forwarding headers
// (X-Forwarded-Host / X-Forwarded-Proto / X-Forwarded-Prefix) and falls back to the request Host,
// the request's TLS state, and the configured base path. If the document is not the expected
// JSON object shape it is returned unchanged.
func patchSwaggerServer(doc string, r *http.Request, configuredBasePath string) string {
	var spec map[string]any
	if err := json.Unmarshal([]byte(doc), &spec); err != nil {
		return doc
	}

	host := firstForwardedValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host != "" {
		spec["host"] = host
	}

	scheme := firstForwardedValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	spec["schemes"] = []string{scheme}

	prefix := strings.TrimRight(firstForwardedValue(r.Header.Get("X-Forwarded-Prefix")), "/")
	if prefix == "" {
		prefix = configuredBasePath
	}
	if prefix == "" {
		prefix = "/"
	}
	spec["basePath"] = prefix

	patched, err := json.Marshal(spec)
	if err != nil {
		return doc
	}
	return string(patched)
}

// firstForwardedValue returns the first entry of a possibly comma-separated forwarding header
// value (proxies may append a list, e.g. "a.example.com, b.internal"), trimmed of whitespace.
func firstForwardedValue(v string) string {
	if v == "" {
		return ""
	}
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
