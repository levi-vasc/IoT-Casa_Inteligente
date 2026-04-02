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
	Auto  bool   `json:"auto"`
}

type Atuador struct {
	id          string
	tipo        string
	estado      bool
	porta       string
	gatewayAddr string
	gatewayConn net.Conn
}

func (a *Atuador) simularAcao() {
	switch a.tipo {
	case "ar":
		if a.estado {
			fmt.Printf("[%s] ❄️  Ar-condicionado LIGADO\n", a.id)
		} else {
			fmt.Printf("[%s] ⭕ Ar-condicionado DESLIGADO\n", a.id)
		}
	case "luz":
		if a.estado {
			fmt.Printf("[%s] 💡 Lâmpada LIGADA\n", a.id)
		} else {
			fmt.Printf("[%s] ⭕ Lâmpada DESLIGADA\n", a.id)
		}
	}
}

func (a *Atuador) conectarGateway() {
	for {
		conn, err := net.Dial("tcp", a.gatewayAddr)
		if err != nil {
			fmt.Printf("[%s] Tentando conectar ao gateway...\n", a.id)
			time.Sleep(2 * time.Second)
			continue
		}
		a.gatewayConn = conn
		fmt.Printf("[%s] Conectado ao gateway\n", a.id)

		conn.Write([]byte(a.id))

		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil || n == 0 {
				break
			}
		}
		fmt.Printf("[%s] Conexão com gateway perdida\n", a.id)
		conn.Close()
		time.Sleep(2 * time.Second)
	}
}

func (a *Atuador) iniciar() {
	listener, err := net.Listen("tcp", ":"+a.porta)
	if err != nil {
		fmt.Printf("[ERRO] %s não conseguiu abrir porta %s: %v\n", a.id, a.porta, err)
		return
	}
	defer listener.Close()

	fmt.Printf("[%s] Aguardando comandos na porta %s...\n", a.id, a.porta)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go a.receberComandos(conn)
	}
}

func (a *Atuador) receberComandos(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("[%s] Gateway/Cliente conectado: %s\n", a.id, conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var cmd Comando
		if err := json.Unmarshal(scanner.Bytes(), &cmd); err != nil {
			continue
		}

		a.estado = cmd.State
		a.simularAcao()
	}
}

func main() {
	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:9000"
	}

	atuadores := []Atuador{
		{id: "ar01", tipo: "ar", estado: false, porta: "9001", gatewayAddr: gatewayAddr},
		{id: "luz01", tipo: "luz", estado: false, porta: "9002", gatewayAddr: gatewayAddr},
	}

	fmt.Println("=== ATUADORES INICIADOS ===")

	for i := range atuadores {
		go atuadores[i].conectarGateway()
		go atuadores[i].iniciar()
	}

	fmt.Println("Atuadores aguardando comandos... Pressione Ctrl+C para parar.")
	select {}
}
