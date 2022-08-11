FROM golang:alpine as builder
ENV CGO_ENABLED=0 \
    GOOS=linux
ADD . /build
WORKDIR /build
RUN go build -a -installsuffix cgo \
        -ldflags '-extldflags "-static"' \
        -o main ./...

FROM scratch
COPY --from=builder /build/main /app/rflink-prom
WORKDIR /app
ENTRYPOINT [ "./rflink-prom" ]
