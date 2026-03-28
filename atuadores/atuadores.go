package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

// ===== COMANDO =====
type Comando struct {
	ID     string `json:"id"`
	Tipo   string `json:"type"`
	Estado bool   `json:"state"`
	Auto   bool   `json:"auto"`
}

// ===== ATUADOR =====
type Atuador struct {
	id     string
	tipo   string
	estado bool
	porta  string
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
	case "porta":
		if a.estado {
			fmt.Printf("[%s] 🔓 Porta ABERTA\n", a.id)
		} else {
			fmt.Printf("[%s] 🔒 Porta FECHADA\n", a.id)
		}
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
	fmt.Printf("[%s] Gateway conectado: %s\n", a.id, conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var cmd Comando
		if err := json.Unmarshal(scanner.Bytes(), &cmd); err != nil {
			fmt.Printf("[ERRO] %s falhou ao ler comando: %v\n", a.id, err)
			continue
		}

		a.estado = cmd.Estado
		a.simularAcao()
	}
}

func main() {
	atuadores := []Atuador{
		{id: "ar01", tipo: "ar", estado: false, porta: "9001"},
		{id: "luz01", tipo: "luz", estado: false, porta: "9002"},
		{id: "porta01", tipo: "porta", estado: false, porta: "9003"},
	}

	done := make(chan struct{})

	for i := range atuadores {
		go atuadores[i].iniciar()
	}

	<-done
}
