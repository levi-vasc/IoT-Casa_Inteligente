package main

import (
	"bufio"
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

type EstadoResposta struct {
	AtuadorID string `json:"atuador_id"`
	Ligado    bool   `json:"ligado"`
}

var (
	cache           = make(map[string]DeviceData)
	cacheMutex      sync.RWMutex
	clients         = make(map[net.Conn]bool)
	clientMux       sync.Mutex
	atuadoresConn   = make(map[string]net.Conn)
	atuadoresMutex  sync.Mutex
	ultimoComando   = make(map[string]time.Time)
	ultimoComandoMu sync.Mutex
	estadoAtuador   = make(map[string]bool)
	estadoMutex     sync.RWMutex
)

var sensorParaAtuador = map[string]string{
	"temp01": "ar01",
	"temp02": "ar02",
	"pres01": "luz01",
	"pres02": "luz02",
}

func main() {
	go startUDPServer(":8081")
	go startTCPServer(":8080")
	go startAtuadoresServer(":9000")
	go startEstadoServer(":9001")

	fmt.Println("[GATEWAY] Sistema iniciado com sucesso")
	fmt.Println("[GATEWAY] Sensores (UDP)           -> :8081")
	fmt.Println("[GATEWAY] Clientes (TCP)           -> :8080")
	fmt.Println("[GATEWAY] Atuadores (TCP)          -> :9000")
	fmt.Println("[GATEWAY] Consulta de estado (TCP) -> :9001")
	select {}
}

// ─── UDP: recebe dados dos sensores ───────────────────────────────────────────

func startUDPServer(port string) {
	addr, _ := net.ResolveUDPAddr("udp", port)
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("[GATEWAY] Erro ao abrir UDP %s: %v\n", port, err)
		return
	}
	defer conn.Close()

	fmt.Printf("[GATEWAY] Escutando Sensores (UDP) na porta %s\n", port)
	buf := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		var data DeviceData
		if err := json.Unmarshal(buf[:n], &data); err != nil {
			fmt.Printf("[GATEWAY] Pacote UDP inválido de %s: %v\n", remoteAddr, err)
			continue
		}

		cacheMutex.Lock()
		cache[data.ID] = data
		cacheMutex.Unlock()

		fmt.Printf("[SENSOR] %s (%s): %v\n", data.ID, data.Type, data.Value)

		processarAutomacao(data)

		payload, _ := json.Marshal(data)
		broadcastToClients(payload)
	}
}

// ─── Lógica de automação ──────────────────────────────────────────────────────

func processarAutomacao(data DeviceData) {
	atuadorID, vinculado := sensorParaAtuador[data.ID]
	if !vinculado {
		return
	}

	switch data.Type {
	case "temperatura":
		if temp, ok := data.Value.(float64); ok && temp > 30 {
			enviarComandoAtuador(atuadorID, true, data.ID)
		}
	case "presenca":
		if presenca, ok := data.Value.(float64); ok {
			enviarComandoAtuador(atuadorID, presenca == 1, data.ID)
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

	chave := fmt.Sprintf("%s_%v", atuadorID, estado)
	agora := time.Now()

	ultimoComandoMu.Lock()
	ultima, existe := ultimoComando[chave]
	if existe && agora.Sub(ultima) < 5*time.Second {
		ultimoComandoMu.Unlock()
		return
	}
	ultimoComando[chave] = agora
	ultimoComandoMu.Unlock()

	cmd := DeviceData{
		ID:    atuadorID,
		Type:  "comando",
		State: estado,
	}
	cmdJSON, _ := json.Marshal(cmd)
	cmdJSON = append(cmdJSON, '\n')

	if _, err := conn.Write(cmdJSON); err != nil {
		fmt.Printf("[AUTOMAÇÃO] Erro ao enviar para %s: %v\n", atuadorID, err)
		atuadoresMutex.Lock()
		delete(atuadoresConn, atuadorID)
		atuadoresMutex.Unlock()
		estadoMutex.Lock()
		delete(estadoAtuador, atuadorID)
		estadoMutex.Unlock()
	} else {
		estadoMutex.Lock()
		estadoAtuador[atuadorID] = estado
		estadoMutex.Unlock()
		publicarEstadoAtuador(atuadorID, estado)
		switch atuadorID {
		case "ar01", "ar02":
			fmt.Printf("[ATUADOR] [%s] ❄️  Ar-condicionado LIGADO\n", atuadorID)
		case "luz01", "luz02":
			fmt.Printf("[ATUADOR] [%s] 💡 Lâmpada LIGADA\n", atuadorID)
		}
	}
}

// ─── TCP: registra atuadores e mantém canal de comandos aberto ────────────────
//
// Protocolo de handshake:
//   1. Atuador conecta em :9000
//   2. Atuador envia seu ID seguido de '\n'  (ex: "ar01\n")
//   3. Gateway registra a conexão no mapa
//   4. A partir daí o gateway escreve comandos JSON+'\n' nessa mesma conn
//   5. O scanner abaixo mantém a goroutine viva e detecta desconexão

func startAtuadoresServer(port string) {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("[GATEWAY] Erro ao abrir porta TCP %s: %v\n", port, err)
		return
	}
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

	scanner := bufio.NewScanner(conn)

	// Primeira linha: ID do atuador
	if !scanner.Scan() {
		fmt.Printf("[GATEWAY] Atuador de %s não enviou ID\n", conn.RemoteAddr())
		return
	}
	atuadorID := scanner.Text()
	if atuadorID == "" {
		fmt.Printf("[GATEWAY] Atuador de %s enviou ID vazio\n", conn.RemoteAddr())
		return
	}

	atuadoresMutex.Lock()
	atuadoresConn[atuadorID] = conn
	atuadoresMutex.Unlock()

	estadoMutex.Lock()
	if _, existe := estadoAtuador[atuadorID]; !existe {
		estadoAtuador[atuadorID] = false
	}
	estadoMutex.Unlock()

	fmt.Printf("[GATEWAY] Atuador registrado: %s (de %s)\n", atuadorID, conn.RemoteAddr())

	// Mantém a goroutine viva — scanner.Scan() retorna false quando a conexão fechar.
	// O gateway envia por conn.Write() em outra goroutine (enviarComandoAtuador).
	for scanner.Scan() {
		// Atuadores não enviam dados além do ID; ignorar qualquer linha extra.
	}

	atuadoresMutex.Lock()
	delete(atuadoresConn, atuadorID)
	atuadoresMutex.Unlock()

	fmt.Printf("[GATEWAY] Atuador desconectado: %s\n", atuadorID)
}

// ─── TCP: consulta de estado (usada pelos sensores) ───────────────────────────

func startEstadoServer(port string) {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("[GATEWAY] Erro ao abrir porta de estado %s: %v\n", port, err)
		return
	}
	fmt.Printf("[GATEWAY] Consulta de estado (TCP) na porta %s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleEstadoQuery(conn)
	}
}

func handleEstadoQuery(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	sensorID := scanner.Text()

	atuadorID, ok := sensorParaAtuador[sensorID]
	if !ok {
		conn.Write([]byte(`{"erro":"sensor nao mapeado"}` + "\n"))
		return
	}

	estadoMutex.RLock()
	ligado := estadoAtuador[atuadorID]
	estadoMutex.RUnlock()

	resp, _ := json.Marshal(EstadoResposta{AtuadorID: atuadorID, Ligado: ligado})
	resp = append(resp, '\n')
	conn.Write(resp)
}

// ─── TCP: recebe comandos dos clientes ────────────────────────────────────────

func startTCPServer(port string) {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("[GATEWAY] Erro ao abrir porta TCP %s: %v\n", port, err)
		return
	}
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

		fmt.Printf("[COMANDO] Recebido de %s → id=%s state=%v\n", conn.RemoteAddr(), cmd.ID, cmd.State)
		enviarComandoAtuador(cmd.ID, cmd.State, "cliente")
	}

	fmt.Printf("[GATEWAY] Cliente desconectado: %s\n", conn.RemoteAddr())
}

func broadcastToClients(data []byte) {
	clientMux.Lock()
	defer clientMux.Unlock()
	for conn := range clients {
		conn.Write(data)
	}
}

func publicarEstadoAtuador(atuadorID string, estado bool) {
	notif, err := json.Marshal(DeviceData{ID: atuadorID, Type: "estado", State: estado})
	if err != nil {
		return
	}
	notif = append(notif, '\n')
	broadcastToClients(notif)
}
