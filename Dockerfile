FROM golang:1.26 AS build

WORKDIR /app

COPY go.mod go.sum ./

COPY cmd /app/cmd
COPY internal /app/internal
COPY pkg /app/pkg

# Install swag v2, generate Swagger 2.0 docs, and build (single layer for smaller image / Sonar Docker rules).
RUN go mod download \
	&& go install github.com/swaggo/swag/v2/cmd/swag@latest \
	&& swag init -g main.go -o docs/ -d ./cmd/fuse,./internal/handlers,./internal/dtos,./internal/workflow,./pkg/workflow \
	&& CGO_ENABLED=0 go build -o bin/fuse /app/cmd/fuse/main.go

## Runnable container

FROM gcr.io/distroless/base-debian11 AS runnable

COPY --from=build /app/bin/fuse /

USER nonroot:nonroot

EXPOSE 9090

CMD ["/fuse", "server", "-l", "debug", "-p", "9090"]
