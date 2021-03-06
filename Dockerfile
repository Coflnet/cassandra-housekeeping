FROM golang:1.17.5-buster as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build .

CMD sh -c /app/cassandra-housekeeping
