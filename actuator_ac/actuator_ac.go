package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

type Comando struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	State bool   `json:"state"`
}

type Atuador struct {
	id          string
	estado      bool
	gatewayAddr string
}

func (a *Atuador) simularAcao() {
	if a.estado {
		fmt.Printf("[%s] ❄️  Ar-condicionado LIGADO\n", a.id)
	} else {
		fmt.Printf("[%s] ⭕ Ar-condicionado DESLIGADO\n", a.id)
	}
}

func (a *Atuador) conectarEEscutar() {
	for {
		conn, err := net.Dial("tcp", a.gatewayAddr)
		if err != nil {
			fmt.Printf("[%s] Tentando conectar ao gateway em %s...\n", a.id, a.gatewayAddr)
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Printf("[%s] Conectado ao gateway\n", a.id)

		// Envia o ID com '\n' como delimitador — o gateway usa bufio.Scanner
		fmt.Fprintf(conn, "%s\n", a.id)

		// Lê comandos JSON que chegam pelo mesmo canal, um por linha
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			linha := scanner.Bytes()
			if len(linha) == 0 {
				continue
			}

			var cmd Comando
			if err := json.Unmarshal(linha, &cmd); err != nil {
				fmt.Printf("[%s] Pacote inválido: %v\n", a.id, err)
				continue
			}

			a.estado = cmd.State
			a.simularAcao()
		}

		fmt.Printf("[%s] Conexão com gateway perdida. Reconectando...\n", a.id)
		conn.Close()
		time.Sleep(2 * time.Second)
	}
}

func main() {
	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:9000"
	}

	atuadorID := os.Getenv("ATUADOR_ID")
	if atuadorID == "" {
		atuadorID = "ar01"
	}

	atuador := &Atuador{
		id:          atuadorID,
		estado:      false,
		gatewayAddr: gatewayAddr,
	}

	fmt.Printf("=== ATUADOR AR-CONDICIONADO [%s] INICIADO ===\n", atuadorID)
	fmt.Printf("Gateway: %s\n", gatewayAddr)
	fmt.Println("Aguardando comandos... Pressione Ctrl+C para parar.")

	atuador.conectarEEscutar()
}
