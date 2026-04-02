FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init sensors
RUN go build -o sensors

CMD ["./sensors"]