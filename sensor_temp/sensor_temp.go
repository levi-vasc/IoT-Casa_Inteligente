package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
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

type EstadoResposta struct {
	AtuadorID string `json:"atuador_id"`
	Ligado    bool   `json:"ligado"`
}

// consultarArLigado abre uma conexão TCP curta com o gateway e pergunta
// se o ar-condicionado vinculado a este sensor está ligado.
// Protocolo: envia "sensorID\n", recebe JSON+"\n".
func consultarArLigado(sensorID, estadoAddr string) bool {
	conn, err := net.DialTimeout("tcp", estadoAddr, 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	fmt.Fprintf(conn, "%s\n", sensorID)

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return false
	}

	var resp EstadoResposta
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return false
	}
	return resp.Ligado
}

func (s *Sensor) SimularTemperatura(gatewayUDPAddr, gatewayEstadoAddr string, intervalo time.Duration) {
	for {
		udpConn, err := net.Dial("udp", gatewayUDPAddr)
		if err != nil {
			fmt.Printf("[%s] Erro ao conectar ao gateway UDP: %v. Tentando novamente...\n", s.ID, err)
			time.Sleep(2 * time.Second)
			continue
		}

		temperatura := 24.0 + rand.Float64()*4.0
		direcao := 1.0
		incremento := 0.2

		fmt.Printf("[%s] Conectado ao gateway %s\n", s.ID, gatewayUDPAddr)

		for {
			arLigado := consultarArLigado(s.ID, gatewayEstadoAddr)

			if arLigado {
				// Ar ligado: aproxima de 20°C, mas mantém leve variação natural.
				if temperatura > 20.0 {
					temperatura -= 0.5
				} else if temperatura < 20.0 {
					temperatura += 0.1
				}
				temperatura += (rand.Float64() - 0.5) * 0.08
				if temperatura < 19.5 {
					temperatura = 19.5
				} else if temperatura > 21.0 {
					temperatura = 21.0
				}
				direcao = 1.0
			} else {
				// Ar desligado: oscila naturalmente entre 18°C e 35°C
				temperatura += incremento * direcao
				if temperatura >= 35 {
					direcao = -1.0
				} else if temperatura <= 18 {
					direcao = 1.0
				}
			}

			s.Value = math.Round(temperatura*100) / 100

			data, err := json.Marshal(s)
			if err != nil {
				fmt.Printf("[%s] Erro ao serializar dado: %v\n", s.ID, err)
				break
			}

			if _, err := udpConn.Write(data); err != nil {
				fmt.Printf("[%s] Erro ao enviar dado: %v\n", s.ID, err)
				break
			}

			arStatus := "desligado"
			if arLigado {
				arStatus = "LIGADO ❄️"
			}
			fmt.Printf("[%s] %.2f°C  (ar: %s)\n", s.ID, temperatura, arStatus)

			time.Sleep(intervalo)
		}

		udpConn.Close()
		time.Sleep(2 * time.Second)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:8081"
	}

	estadoAddr := os.Getenv("GATEWAY_ESTADO_ADDR")
	if estadoAddr == "" {
		host, _, _ := net.SplitHostPort(gatewayAddr)
		if host == "" {
			host = "localhost"
		}
		estadoAddr = host + ":9001"
	}

	sensorID := os.Getenv("SENSOR_ID")
	if sensorID == "" {
		sensorID = "temp01"
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
		Type: "temperatura",
	}

	fmt.Printf("=== SENSOR TEMPERATURA [%s] INICIADO ===\n", sensorID)
	fmt.Printf("Gateway UDP:    %s\n", gatewayAddr)
	fmt.Printf("Gateway Estado: %s\n", estadoAddr)
	fmt.Printf("Intervalo:      %dms\n", intervalo)

	sensor.SimularTemperatura(gatewayAddr, estadoAddr, time.Duration(intervalo)*time.Millisecond)
}
