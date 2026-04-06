FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init sensor_pres
RUN go build -o sensor_pres

CMD ["./sensor_pres"]