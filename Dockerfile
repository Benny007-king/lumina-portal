# Lumina licensing portal — pure-Go, no CGO → tiny static distroless image.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/portal .

# busybox stage: create a /data dir the nonroot user (65532) can write to
# (distroless has no shell/mkdir, and SQLite needs a writable data dir).
FROM busybox:1.36 AS perms
RUN mkdir -p /data && chown -R 65532:65532 /data

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/portal /portal
COPY --from=perms --chown=65532:65532 /data /data
ENV PORTAL_ADDR=0.0.0.0:8090 DATA_DIR=/data
EXPOSE 8090
USER 65532:65532
ENTRYPOINT ["/portal"]
