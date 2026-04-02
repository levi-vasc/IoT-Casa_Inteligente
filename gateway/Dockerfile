FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init gateway
RUN go build -o gateway

CMD ["./gateway"]