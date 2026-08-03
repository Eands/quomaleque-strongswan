FROM golang:alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/vpn-radius ./main/

FROM golang:alpine AS debug

RUN apk add --no-cache gcc musl-dev
RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -gcflags="all=-N -l" -o /app/vpn-radius ./main/

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates sqlite-libs wget

COPY --from=builder /app/vpn-radius /usr/local/bin/vpn-radius
COPY --from=builder /app/web /app/web

RUN mkdir -p /data

EXPOSE 8080
EXPOSE 1812/udp
EXPOSE 1813/udp

ENTRYPOINT ["/usr/local/bin/vpn-radius"]
