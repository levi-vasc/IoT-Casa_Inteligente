package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Erro ao conectar ao Gateway:", err)
		return
	}
	defer conn.Close()

	fmt.Println("=== CLIENTE ROTA DAS COISAS ===")
	fmt.Println("Comandos: ligar <id> | desligar <id>")

	// Thread para receber dados em tempo real (Monitoramento) [cite: 23]
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Printf("\r[NOTIFICAÇÃO] %s\n> ", scanner.Text())
		}
	}()

	// Loop principal para envio de comandos
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		parts := strings.Split(text, " ")

		if len(parts) < 2 {
			continue
		}

		acao := parts[0]
		id := parts[1]
		estado := (acao == "ligar")

		cmd, _ := json.Marshal(map[string]interface{}{
			"id":    id,
			"state": estado,
		})
		conn.Write(cmd)
	}
}
