package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"time"
)

type Sensor struct {
	ID    string      `json:"id"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (s *Sensor) SimularTemperatura(gatewayAddr string, intervalo time.Duration) {
	conn, _ := net.Dial("udp", gatewayAddr)
	defer conn.Close()

	temperatura := 20.0
	direcao := 1.0
	incremento := 0.2

	for {
		s.Value = math.Round(temperatura*100) / 100

		data, _ := json.Marshal(s)
		conn.Write(data)
		fmt.Printf("[%s] Enviado: %.2f°C\n", s.ID, s.Value)

		temperatura += (incremento * direcao)

		if temperatura >= 35 {
			direcao = -1.0
		} else if temperatura <= 18 {
			direcao = 1.0
		}

		time.Sleep(intervalo)
	}
}

func (s *Sensor) SimularPresenca(gatewayAddr string, intervalo time.Duration) {
	conn, _ := net.Dial("udp", gatewayAddr)
	defer conn.Close()

	for {
		// Gera novo valor aleatório
		s.Value = rand.Intn(2)

		status := "ausente"
		if s.Value.(int) == 1 {
			status = "presente"
		}
		fmt.Printf("[%s] Novo valor gerado: %s\n", s.ID, status)

		// Repete o mesmo valor por 10 segundos no mesmo ritmo do intervalo
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			data, _ := json.Marshal(s)
			conn.Write(data)
			fmt.Printf("[%s] Enviado: %s\n", s.ID, status)
			time.Sleep(intervalo)
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:8081"
	}

	fmt.Println("=== SENSORES INICIADOS ===")

	go (&Sensor{ID: "temp01", Type: "temperatura"}).SimularTemperatura(gatewayAddr, 1*time.Second)
	go (&Sensor{ID: "temp02", Type: "temperatura"}).SimularTemperatura(gatewayAddr, 1*time.Second)

	go (&Sensor{ID: "pres01", Type: "presenca"}).SimularPresenca(gatewayAddr, 1*time.Second)
	go (&Sensor{ID: "pres02", Type: "presenca"}).SimularPresenca(gatewayAddr, 1*time.Second)

	fmt.Println("Sensores em execução... Pressione Ctrl+C para parar.")
	select {}
}
