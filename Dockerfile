FROM golang:1.25 AS build

WORKDIR /app

COPY go.mod go.sum ./

COPY cmd /app/cmd
COPY internal /app/internal
COPY pkg /app/pkg

RUN go mod download

# Install swag and generate swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/fuse/main.go -o docs/

RUN CGO_ENABLED=0 go build -o bin/fuse /app/cmd/fuse/main.go

## Runnable container

FROM gcr.io/distroless/base-debian11 AS runnable

COPY --from=build /app/bin/fuse /

USER nonroot:nonroot

EXPOSE 9090

CMD ["/fuse", "server", "-l", "debug", "-p", "9090"]
