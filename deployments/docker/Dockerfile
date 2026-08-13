# syntax=docker/dockerfile:1

FROM golang:1.26.5-bookworm AS build

WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

ARG SERVICE=documentviewer
ARG VERSION=dev

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/app \
      ./cmd/${SERVICE}

FROM alpine:3.21

# Logger writes JSON to logs/<service>.log (cwd /). nonroot cannot mkdir /.
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 65532 nonroot \
    && mkdir -p /logs \
    && chown nonroot:nonroot /logs \
    && chmod 0750 /logs

COPY --from=build /out/app /app

USER nonroot:nonroot

EXPOSE 8000 9100 9101

ENTRYPOINT ["/app"]
