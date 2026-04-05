FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod init sensor_temp
RUN go build -o sensor_temp

CMD ["./sensor_temp"]