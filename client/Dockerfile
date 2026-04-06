FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init client
RUN go build -o client

CMD ["./client"]