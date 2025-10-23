# Swagger Documentation

This directory contains auto-generated Swagger/OpenAPI documentation for the FUSE Workflow Engine API.

## Generated Files

The following files are auto-generated and should not be edited manually:

- `docs.go` - Go code for embedding Swagger documentation
- `swagger.json` - OpenAPI specification in JSON format
- `swagger.yaml` - OpenAPI specification in YAML format

These files are automatically generated from code annotations and are excluded from version control.

## Generating Documentation

To regenerate the Swagger documentation after making changes to API endpoints or handler annotations:

```bash
make swagger
```

Or run directly:

```bash
swag init -g cmd/fuse/main.go -o docs/
```

## Viewing Documentation

Once the server is running, visit:

```
http://localhost:9090/docs
```

Or the full path:

```
http://localhost:9090/docs/index.html
```

Replace `localhost:9090` with your actual server host and port.

## Updating Documentation

To add or update API documentation:

1. Add Swagger annotations to handler methods in `internal/handlers/`
2. Run `make swagger` to regenerate documentation
3. Restart the server to see changes

## Swagger Annotation Format

Example handler with Swagger annotations:

```go
// HandleGet handles the GET request
// @Summary Get resource
// @Description Retrieve a resource by ID
// @Tags resources
// @Accept json
// @Produce json
// @Param id path string true "Resource ID"
// @Success 200 {object} ResourceResponse
// @Failure 400 {object} BadRequestError
// @Failure 404 {object} NotFoundError
// @Failure 500 {object} InternalServerErrorResponse
// @Router /v1/resources/{id} [get]
func (h *Handler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
    // implementation
}
```

## References

- [Swaggo Documentation](https://github.com/swaggo/swag)
- [OpenAPI Specification](https://swagger.io/specification/)
