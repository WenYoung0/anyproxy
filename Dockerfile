FROM golang:1.25.1-alpine AS builder
WORKDIR /app
COPY . .

RUN rm -rf dist && \
    mkdir -p dist && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOMIPS=softfloat \
    go build -trimpath -ldflags="-s -w" -o /app/dist/anyproxy cmd/anyproxy/*.go


FROM scratch AS final
WORKDIR /app

COPY --from=builder /app/dist/anyproxy anyproxy
COPY --from=builder /app/example.yaml config.yaml

EXPOSE 6646

CMD ["./anyproxy","run","-c","/app/config.yaml"]