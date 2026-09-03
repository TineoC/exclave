FROM golang:1.24-alpine AS build
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /svc main.go

FROM scratch
# Baked in at build time so each service is a distinct artifact with its own
# digest. Two services built from identical source would otherwise be the same
# image — correct for content-addressing, confusing in a demo.
ARG SERVICE=svc
ARG VERSION=0.0.0
LABEL org.opencontainers.image.title="$SERVICE"
LABEL org.opencontainers.image.version="$VERSION"
ENV SERVICE_NAME=$SERVICE
ENV SERVICE_VERSION=$VERSION
COPY --from=build /svc /svc
USER 65532:65532
ENTRYPOINT ["/svc"]
