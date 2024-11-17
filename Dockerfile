FROM golang:latest as builder

WORKDIR /app

COPY go.mod go.sum ./
COPY . .
RUN go mod download

RUN CGO_ENABLED=1 go build -o bskyfeed .

FROM alpine:latest

RUN apk --no-cache add ca-certificates libgcc

WORKDIR /root/
COPY --from=builder /app/bskyfeed .

CMD ["./bskyfeed"]
