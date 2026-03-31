package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
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

	// Valor inicial e direção da variação
	temperatura := 20.0
	direcao := 1.0 // 1.0 para aumentar, -1.0 para diminuir
	incremento := 0.2

	for {
		s.Value = math.Round(temperatura*100) / 100

		data, _ := json.Marshal(s)
		conn.Write(data)
		fmt.Printf("[%s] Enviado: %.2f°C\n", s.ID, s.Value)

		// Atualizar temperatura
		temperatura += (incremento * direcao)

		// Inverter direção ao atingir limites (18°C a 35°C)
		if temperatura >= 35 {
			direcao = -1.0
		} else if temperatura <= 18 {
			direcao = 1.0
		}

		time.Sleep(intervalo)
	}
}

func (s *Sensor) SimularPresenca(gatewayAddr string) {
	conn, _ := net.Dial("udp", gatewayAddr)
	defer conn.Close()

	for {
		// Esperar 15 segundos antes de gerar novo valor
		time.Sleep(15 * time.Second)

		// Presença: 0 (ausente) ou 1 (presente)
		s.Value = rand.Intn(2)

		data, _ := json.Marshal(s)
		conn.Write(data)

		status := "ausente"
		if s.Value.(int) == 1 {
			status = "presente"
		}
		fmt.Printf("[%s] Enviado: %s\n", s.ID, status)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("=== SENSORES INICIADOS ===")

	// Sensores de Temperatura - aumenta/diminui 0.2°C a cada 1 segundo
	go (&Sensor{ID: "temp01", Type: "temperatura"}).SimularTemperatura("localhost:8081", 1*time.Second)
	go (&Sensor{ID: "temp02", Type: "temperatura"}).SimularTemperatura("localhost:8081", 1*time.Second)

	// Sensores de Presença - gera novo valor a cada 15 segundos
	go (&Sensor{ID: "pres01", Type: "presenca"}).SimularPresenca("localhost:8081")
	go (&Sensor{ID: "pres02", Type: "presenca"}).SimularPresenca("localhost:8081")

	fmt.Println("Sensores em execução... Pressione Ctrl+C para parar.")
	select {}
}
