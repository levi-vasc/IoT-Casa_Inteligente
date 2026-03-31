package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type DeviceData struct {
	ID    string      `json:"id"`
	Type  string      `json:"type"`
	Value interface{} `json:"value,omitempty"`
	State bool        `json:"state,omitempty"`
}

var (
	cache             = make(map[string]DeviceData)
	cacheMutex        sync.RWMutex
	clients           = make(map[net.Conn]bool)
	clientMux         sync.Mutex
	atuadoresConn     = make(map[string]net.Conn) // Conexões com atuadores
	atuadoresMutex    sync.Mutex
	ultimoComandoTemp = make(map[string]time.Time) // Throttling de comandos
	ultimoComandoPres = make(map[string]time.Time)
)

func main() {
	// Thread para Telemetria (UDP) - Sensores
	go startUDPServer(":8081")

	// Thread para Controle e Visualização (TCP) - Clientes
	go startTCPServer(":8080")

	// Thread para conexão com Atuadores (TCP)
	go startAtuadoresServer(":9000")

	fmt.Println("[GATEWAY] Sistema iniciado com sucesso")
	select {}
}

func startUDPServer(port string) {
	addr, _ := net.ResolveUDPAddr("udp", port)
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	fmt.Printf("[GATEWAY] Escutando Sensores (UDP) na porta %s\n", port)
	buf := make([]byte, 1024)

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		var data DeviceData
		if err := json.Unmarshal(buf[:n], &data); err == nil {
			cacheMutex.Lock()
			cache[data.ID] = data
			cacheMutex.Unlock()

			// Exibir sensor recebido
			fmt.Printf("[SENSOR] %s (%s): %v\n", data.ID, data.Type, data.Value)

			// Processar automação
			processarAutomacao(data)

			// Repassar para clientes
			broadcastToClients(buf[:n])
		}
	}
}

func processarAutomacao(data DeviceData) {
	switch data.Type {
	case "temperatura":
		// Se temperatura > 30°C, ligar ar-condicionado
		if temp, ok := data.Value.(float64); ok && temp > 30 {
			enviarComandoAtuador("ar01", true, data.ID)
		}

	case "presenca":
		// Se presença = 1, ligar lâmpada
		if presenca, ok := data.Value.(float64); ok && presenca == 1 {
			enviarComandoAtuador("luz01", true, data.ID)
		}
	}
}

func enviarComandoAtuador(atuadorID string, estado bool, sensorID string) {
	atuadoresMutex.Lock()
	conn, exists := atuadoresConn[atuadorID]
	atuadoresMutex.Unlock()

	if !exists || conn == nil {
		fmt.Printf("[AUTOMAÇÃO] Atuador %s não está conectado\n", atuadorID)
		return
	}

	// Throttling: não enviar comando muito frequentemente
	chave := atuadorID + "_" + fmt.Sprintf("%v", estado)
	agora := time.Now()

	if ultima, exists := ultimoComandoTemp[chave]; exists {
		if agora.Sub(ultima) < 5*time.Second {
			return // Ignorar comandos muito frequentes
		}
	}

	ultimoComandoTemp[chave] = agora

	cmd := DeviceData{
		ID:    atuadorID,
		Type:  "comando",
		State: estado,
	}
	cmdJSON, _ := json.Marshal(cmd)
	cmdJSON = append(cmdJSON, '\n')

	if _, err := conn.Write(cmdJSON); err == nil {
		fmt.Printf("[AUTOMAÇÃO] ✓ %s ativado por %s (estado: %v)\n", atuadorID, sensorID, estado)
	}
}

func startTCPServer(port string) {
	ln, _ := net.Listen("tcp", port)
	fmt.Printf("[GATEWAY] Escutando Clientes (TCP) na porta %s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		clientMux.Lock()
		clients[conn] = true
		clientMux.Unlock()

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer func() {
		clientMux.Lock()
		delete(clients, conn)
		clientMux.Unlock()
		conn.Close()
	}()

	fmt.Printf("[GATEWAY] Novo cliente conectado: %s\n", conn.RemoteAddr())

	decoder := json.NewDecoder(conn)
	for {
		var cmd DeviceData
		if err := decoder.Decode(&cmd); err != nil {
			break
		}

		fmt.Printf("[COMANDO] Recebido para %s: %v\n", cmd.ID, cmd.State)
		enviarComandoAtuador(cmd.ID, cmd.State, "cliente")

		broadcastToClients([]byte(fmt.Sprintf("{\"id\":\"%s\",\"state\":%v}\n", cmd.ID, cmd.State)))
	}
}

func startAtuadoresServer(port string) {
	ln, _ := net.Listen("tcp", port)
	fmt.Printf("[GATEWAY] Escutando Atuadores (TCP) na porta %s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go handleAtuador(conn)
	}
}

func handleAtuador(conn net.Conn) {
	defer conn.Close()

	// Ler ID do atuador
	buf := make([]byte, 256)
	n, _ := conn.Read(buf)
	atuadorID := string(buf[:n])

	atuadoresMutex.Lock()
	atuadoresConn[atuadorID] = conn
	atuadoresMutex.Unlock()

	fmt.Printf("[GATEWAY] Atuador conectado: %s\n", atuadorID)

	// Manter conexão aberta
	for {
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			break
		}
	}

	atuadoresMutex.Lock()
	delete(atuadoresConn, atuadorID)
	atuadoresMutex.Unlock()

	fmt.Printf("[GATEWAY] Atuador desconectado: %s\n", atuadorID)
}

func broadcastToClients(data []byte) {
	clientMux.Lock()
	defer clientMux.Unlock()
	for conn := range clients {
		conn.Write(data)
	}
}
