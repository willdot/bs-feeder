FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY . ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o bskyfeed .

FROM alpine:latest

RUN apk --no-cache add ca-certificates libgcc

WORKDIR /root
COPY --from=builder /app/bskyfeed .
CMD ["/bskyfeed"]
