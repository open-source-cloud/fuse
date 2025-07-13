FROM golang:1.24 as build

WORKDIR /app

COPY go.mod go.sum ./

COPY app /app/app
COPY cmd /app/cmd
COPY internal /app/internal
COPY pkg /app/pkg
COPY logging /app/logging
COPY tests /app/tests

RUN go mod download
RUN CGO_ENABLED=0 go build -o bin/fuse /app/cmd/fuse/main.go

## Runnable container

FROM gcr.io/distroless/base-debian11 as runnable

COPY --from=build /app/bin/fuse /

EXPOSE 9090

CMD ["/fuse", "server", "-l", "debug", "-p", "9090"]