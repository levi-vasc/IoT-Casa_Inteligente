FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init actuator_light
RUN go build -o actuator_light

CMD ["./actuator_light"]