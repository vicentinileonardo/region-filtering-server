FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod .
COPY main.go .

COPY data/ data/

RUN go build -o server

EXPOSE 8080
CMD ["./server"]