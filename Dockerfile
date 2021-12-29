FROM golang:1.17.5-buster as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build .

FROM alpine:3.14

COPY --from=builder /app/cassandra-housekeeping /usr/local/bin/cassandra-housekeeping

CMD cassandra-housekeeping