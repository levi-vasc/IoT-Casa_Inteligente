FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init actuator_ac
RUN go build -o actuator_ac

CMD ["./actuator_ac"]