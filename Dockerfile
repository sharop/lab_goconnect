# Multi stage build

#Build stage
FROM golang:1.17-alpine AS build
WORKDIR /go/src/calli
copy . .
RUN CGO_ENABLED=0 go build -o /go/bin/calli ./cmd/calli

#Run stage
FROM scratch
COPY --from=build /go/bin/calli /bin/calli
ENTRYPOINT ["/bin/calli"]
