ARG TARGET=server

FROM golang:1.27.1-alpine AS build
WORKDIR /src
ARG TARGET=server
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/. .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app ./cmd/${TARGET}

FROM gcr.io/distroless/static-debian12:nonroot AS runtime-server
COPY --from=build /app /server
ENTRYPOINT ["/server"]

FROM runtime-server AS runtime-jobworker

FROM gcr.io/distroless/static-debian12:nonroot AS runtime-worker
COPY --from=build /app /worker
ENTRYPOINT ["/worker"]

ARG TARGET=server
FROM runtime-${TARGET} AS final
