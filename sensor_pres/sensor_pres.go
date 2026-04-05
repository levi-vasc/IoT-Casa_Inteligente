package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

type Sensor struct {
	ID    string      `json:"id"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (s *Sensor) SimularPresenca(gatewayAddr string, intervalo time.Duration) {
	for {
		conn, err := net.Dial("udp", gatewayAddr)
		if err != nil {
			fmt.Printf("[%s] Erro ao conectar ao gateway: %v. Tentando novamente...\n", s.ID, err)
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Printf("[%s] Conectado ao gateway %s\n", s.ID, gatewayAddr)

		for {
			s.Value = rand.Intn(2)

			status := "ausente"
			if s.Value.(int) == 1 {
				status = "presente"
			}
			fmt.Printf("[%s] Novo valor gerado: %s\n", s.ID, status)

			deadline := time.Now().Add(10 * time.Second)
			errEnvio := false

			for time.Now().Before(deadline) {
				data, err := json.Marshal(s)
				if err != nil {
					fmt.Printf("[%s] Erro ao serializar dado: %v\n", s.ID, err)
					errEnvio = true
					break
				}

				_, err = conn.Write(data)
				if err != nil {
					fmt.Printf("[%s] Erro ao enviar dado: %v\n", s.ID, err)
					errEnvio = true
					break
				}

				fmt.Printf("[%s] Enviado: %s\n", s.ID, status)
				time.Sleep(intervalo)
			}

			if errEnvio {
				break
			}
		}

		conn.Close()
		time.Sleep(2 * time.Second)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Configuração via variáveis de ambiente
	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:8081"
	}

	sensorID := os.Getenv("SENSOR_ID")
	if sensorID == "" {
		sensorID = "pres01"
	}

	intervaloStr := os.Getenv("INTERVALO_MS")
	intervalo := 1000
	if intervaloStr != "" {
		if v, err := strconv.Atoi(intervaloStr); err == nil {
			intervalo = v
		}
	}

	sensor := &Sensor{
		ID:   sensorID,
		Type: "presenca",
	}

	fmt.Printf("=== SENSOR PRESENÇA [%s] INICIADO ===\n", sensorID)
	fmt.Printf("Gateway: %s | Intervalo: %dms\n", gatewayAddr, intervalo)

	sensor.SimularPresenca(gatewayAddr, time.Duration(intervalo)*time.Millisecond)
}
