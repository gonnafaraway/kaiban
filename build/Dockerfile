FROM golang:1.24-bookworm AS build
WORKDIR /src
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates git && rm -rf /var/lib/apt/lists/*
COPY --from=build /api /api
COPY migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/api"]
