FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /webapp ./cmd/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /webapp /webapp
COPY templates/ /templates/
COPY static/ /static/

EXPOSE 443 80

ENTRYPOINT ["/webapp"]
