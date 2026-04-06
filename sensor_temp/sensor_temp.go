package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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

func (s *Sensor) SimularTemperatura(gatewayUDPAddr string, intervalo time.Duration) {
	for {
		udpConn, err := net.Dial("udp", gatewayUDPAddr)
		if err != nil {
			fmt.Printf("[%s] Erro ao conectar ao gateway UDP: %v. Tentando novamente...\n", s.ID, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Começa numa faixa realista
		temperatura := 24.0 + rand.Float64()*2.0 // 24..26

		// Limites de oscilação natural
		minT := 18.0
		maxT := 35.0

		fmt.Printf("[%s] Conectado ao gateway %s\n", s.ID, gatewayUDPAddr)

		for {
			// Delta pequeno para evitar saltos bruscos
			delta := (rand.Float64()*2 - 1) * 0.5 // -0.3..+0.3

			// “Força” suave para voltar ao meio quando estiver nos extremos
			if temperatura > 33.0 {
				delta -= 0.2
			} else if temperatura < 20.0 {
				delta += 0.2
			}

			temperatura += delta

			// Clamp
			if temperatura < minT {
				temperatura = minT
			} else if temperatura > maxT {
				temperatura = maxT
			}

			// Arredonda para 2 casas
			s.Value = float64(int(temperatura*100)) / 100

			data, err := json.Marshal(s)
			if err != nil {
				fmt.Printf("[%s] Erro ao serializar dado: %v\n", s.ID, err)
				break
			}

			if _, err := udpConn.Write(data); err != nil {
				fmt.Printf("[%s] Erro ao enviar dado: %v\n", s.ID, err)
				break
			}

			fmt.Printf("[%s] %.2f°C\n", s.ID, temperatura)
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

	sensorID := os.Getenv("SENSOR_ID")
	if sensorID == "" {
		sensorID = "temp01"
	}

	intervalo := 1000

	sensor := &Sensor{
		ID:   sensorID,
		Type: "temperatura",
	}

	fmt.Printf("=== SENSOR TEMPERATURA [%s] INICIADO ===\n", sensorID)
	fmt.Printf("Gateway UDP:    %s\n", gatewayAddr)
	fmt.Printf("Intervalo:      %dms\n", intervalo)

	sensor.SimularTemperatura(gatewayAddr, time.Duration(intervalo)*time.Millisecond)
}
