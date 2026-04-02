FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init actuators
RUN go build -o actuators

CMD ["./actuators"]