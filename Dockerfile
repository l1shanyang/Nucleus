# ---- 编译阶段 ----
ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w \
      -X nucleus/internal/version.Version=${VERSION} \
      -X nucleus/internal/version.Commit=${COMMIT} \
      -X nucleus/internal/version.BuildTime=${BUILD_TIME}" \
    -o /bin/api ./cmd/api

# ---- 运行阶段 ----
FROM scratch

COPY --from=builder /bin/api /api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/api"]
