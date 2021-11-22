# Multi stage build

#Build stage
FROM golang:1.17-alpine AS build
WORKDIR /go/src/calli
copy . .
RUN CGO_ENABLED=0 go build -o /go/bin/calli ./cmd/calli
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.6 && \
    wget -qO/go/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && chmod +x /go/bin/grpc_health_probe


#Run stage
FROM scratch
COPY --from=build /go/bin/calli /bin/calli
COPY --from=build /go/bin/grpc_health_probe /bin/grpc_health_probe
ENTRYPOINT ["/bin/calli"]
