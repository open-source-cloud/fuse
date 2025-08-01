FROM golang:1.24 AS build

WORKDIR /app

COPY go.mod go.sum ./

COPY cmd /app/cmd
COPY internal /app/internal
COPY pkg /app/pkg

RUN go mod download
RUN CGO_ENABLED=0 go build -o bin/fuse /app/cmd/fuse/main.go

## Runnable container

FROM gcr.io/distroless/base-debian11 AS runnable

COPY --from=build /app/bin/fuse /

USER nonroot:nonroot

EXPOSE 9090

CMD ["/fuse", "server", "-l", "debug", "-p", "9090"]
