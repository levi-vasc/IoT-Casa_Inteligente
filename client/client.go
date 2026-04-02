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
	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:8080"
	}

	conn, err := net.Dial("tcp", gatewayAddr)
	if err != nil {
		fmt.Println("Erro ao conectar ao Gateway:", err)
		return
	}
	defer conn.Close()

	fmt.Println("=== CLIENTE ROTA DAS COISAS ===")
	fmt.Println("Comandos: ligar <id> | desligar <id>")

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Printf("\r[NOTIFICAÇÃO] %s\n> ", scanner.Text())
		}
	}()

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
