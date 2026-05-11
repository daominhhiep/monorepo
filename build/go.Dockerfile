# Multi-stage Go build, parameterised by BIN_PATH and BIN_NAME.
# Usage (via docker buildx bake or docker build):
#   docker build -f build/go.Dockerfile \
#     --build-arg BIN_PATH=./services/user/cmd/user \
#     --build-arg BIN_NAME=user \
#     -t base/user:dev .
ARG GO_VERSION=1.25.10
FROM golang:${GO_VERSION}-alpine3.22 AS builder
ARG BIN_PATH
ARG BIN_NAME
ARG VERSION=dev
ARG COMMIT=unknown
ENV CGO_ENABLED=0 GOFLAGS="-trimpath"
WORKDIR /src
COPY go.work go.work.sum* ./
COPY gen ./gen
COPY pkg ./pkg
COPY services ./services
COPY apps ./apps
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go work sync && \
    go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
        -o /out/${BIN_NAME} ${BIN_PATH}

FROM gcr.io/distroless/static-debian12:nonroot
ARG BIN_NAME
COPY --from=builder /out/${BIN_NAME} /app/bin
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/bin"]
