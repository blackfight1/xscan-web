# Build stage
FROM golang:1.21-alpine AS build
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o xscan-web .

# Production stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates wget unzip
WORKDIR /app

COPY --from=build /app/xscan-web .

# Create directories
RUN mkdir -p data results tools static xscan

EXPOSE 8080
CMD ["./xscan-web"]
