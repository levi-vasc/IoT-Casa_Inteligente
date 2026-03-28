package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"
)

type SensorData struct {
	ID    string  `json:"id"`
	Tipo  string  `json:"type"`
	Valor float64 `json:"value"`
}

func gerarValores(tipo string) float64 {
	switch tipo {
	case "temperatura":
		return 20 + rand.Float64()*10
	case "umidade":
		return 40 + rand.Float64()*40
	case "luminosidade":
		return rand.Float64() * 1000
	case "presenca":
		if rand.Intn(2) == 0 {
			return 0
		}
		return 1
	}
	return 0
}

func iniciarSensor(sensor SensorData, intervalo time.Duration) {
	conn, err := net.Dial("udp", "localhost:8081")
	if err != nil {
		fmt.Printf("[ERRO] Sensor %s não conseguiu conectar: %v\n", sensor.ID, err)
		return
	}
	defer conn.Close()

	for {
		sensor.Valor = gerarValores(sensor.Tipo)

		dados, err := json.Marshal(sensor)
		if err != nil {
			fmt.Printf("[ERRO] Sensor %s falhou ao serializar: %v\n", sensor.ID, err)
			continue
		}

		conn.Write(dados)
		fmt.Printf("[ENVIADO] %s | tipo: %-12s | valor: %.2f\n", sensor.ID, sensor.Tipo, sensor.Valor)
		time.Sleep(intervalo)
	}
}

func iniciarServidor() {
	addr, _ := net.ResolveUDPAddr("udp", ":8081")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("[ERRO] Servidor: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("[SERVIDOR] Escutando na porta 8081...")

	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		var sensor SensorData
		if err := json.Unmarshal(buf[:n], &sensor); err != nil {
			fmt.Printf("[ERRO] Falha ao deserializar: %v\n", err)
			continue
		}

		fmt.Printf("[RECEBIDO] %s | tipo: %-12s | valor: %.2f\n", sensor.ID, sensor.Tipo, sensor.Valor)
	}
}

func main() {
	go iniciarServidor()
	time.Sleep(100 * time.Millisecond) // aguarda servidor subir

	sensors := []struct {
		dados     SensorData
		intervalo time.Duration
	}{
		{SensorData{"temp01", "temperatura", 0}, 1 * time.Second},
		{SensorData{"umi01", "umidade", 0}, 2 * time.Second},
		{SensorData{"luz01", "luminosidade", 0}, 500 * time.Millisecond},
		{SensorData{"presenca01", "presenca", 0}, 3 * time.Second},
	}

	for _, s := range sensors {
		go iniciarSensor(s.dados, s.intervalo)
	}

	select {}
}
