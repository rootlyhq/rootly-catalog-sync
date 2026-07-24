FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /rootly-catalog-sync ./cmd/rootly-catalog-sync

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /rootly-catalog-sync /usr/local/bin/
ENTRYPOINT ["rootly-catalog-sync"]
