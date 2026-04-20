FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/ragbot .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/ragbot /ragbot
EXPOSE 8080
ENTRYPOINT ["/ragbot"]
