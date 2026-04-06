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

var (
	cache          = make(map[string]DeviceData)
	cacheMutex     sync.RWMutex
	clients        = make(map[net.Conn]bool)
	clientMux      sync.Mutex
	atuadoresConn  = make(map[string]net.Conn)
	atuadoresMutex sync.Mutex

	ultimoComando   = make(map[string]time.Time)
	ultimoComandoMu sync.Mutex

	estadoAtuador = make(map[string]bool)
	estadoMutex   sync.RWMutex

	// Override de luz via cliente (bloqueia automação por um tempo)
	luzOverrideAte   = make(map[string]time.Time) // ex: "luz01" -> now+5s
	luzOverrideMutex sync.Mutex
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

	fmt.Println("[GATEWAY] Sistema iniciado com sucesso")
	fmt.Println("[GATEWAY] Sensores (UDP)  -> :8081")
	fmt.Println("[GATEWAY] Clientes (TCP)  -> :8080")
	fmt.Println("[GATEWAY] Atuadores (TCP) -> :9000")
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
		temp, ok := data.Value.(float64)
		if !ok {
			return
		}

		// Histerese:
		// - Liga quando temp >= 26
		// - Desliga quando temp <= 20
		// - Entre 20 e 26, mantém estado atual (não envia comando)
		estadoMutex.RLock()
		ligado := estadoAtuador[atuadorID]
		estadoMutex.RUnlock()

		if !ligado && temp >= 26.0 {
			enviarComandoAtuador(atuadorID, true, data.ID)
		} else if ligado && temp <= 20.0 {
			enviarComandoAtuador(atuadorID, false, data.ID)
		}

	case "presenca":
		presencaF, ok := data.Value.(float64)
		if !ok {
			fmt.Printf("[GATEWAY] Valor de presença inválido (%T) para %s: %v\n", data.Value, data.ID, data.Value)
			return
		}

		ligar := presencaF == 1

		// Se cliente comandou luz recentemente, ignora automação por 5s
		if (atuadorID == "luz01" || atuadorID == "luz02") && luzEmOverride(atuadorID) {
			return
		}

		enviarComandoAtuador(atuadorID, ligar, data.ID)
	}
}

func enviarComandoAtuador(atuadorID string, estado bool, origem string) {
	atuadoresMutex.Lock()
	conn, exists := atuadoresConn[atuadorID]
	atuadoresMutex.Unlock()

	if !exists || conn == nil {
		fmt.Printf("[AUTOMAÇÃO] Atuador %s não está conectado\n", atuadorID)
		return
	}

	// Anti-spam: evita repetir o mesmo comando num intervalo curto
	chave := fmt.Sprintf("%s_%v", atuadorID, estado)
	agora := time.Now()

	ultimoComandoMu.Lock()
	if ultima, existe := ultimoComando[chave]; existe && agora.Sub(ultima) < 5*time.Second {
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

		return
	}

	// Atualiza estado e notifica clientes
	estadoMutex.Lock()
	estadoAtuador[atuadorID] = estado
	estadoMutex.Unlock()

	publicarEstadoAtuador(atuadorID, estado)
	logEstadoAtuador(atuadorID, estado, origem)
}

func logEstadoAtuador(atuadorID string, estado bool, origem string) {
	sufixo := "DESLIGADO"
	if estado {
		sufixo = "LIGADO"
	}

	switch atuadorID {
	case "ar01", "ar02":
		fmt.Printf("[ATUADOR] [%s] ❄️  Ar-condicionado %s (origem: %s)\n", atuadorID, sufixo, origem)
	case "luz01", "luz02":
		fmt.Printf("[ATUADOR] [%s] 💡 Lâmpada %s (origem: %s)\n", atuadorID, sufixo, origem)
	default:
		fmt.Printf("[ATUADOR] [%s] %s (origem: %s)\n", atuadorID, sufixo, origem)
	}
}

// ─── TCP: registra atuadores e mantém canal de comandos aberto ────────────────

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
	for scanner.Scan() {
		// Atuadores não enviam dados além do ID; ignorar qualquer linha extra.
	}

	atuadoresMutex.Lock()
	delete(atuadoresConn, atuadorID)
	atuadoresMutex.Unlock()

	fmt.Printf("[GATEWAY] Atuador desconectado: %s\n", atuadorID)
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

		// Se for luz, prioridade ao cliente por 5s
		if cmd.ID == "luz01" || cmd.ID == "luz02" {
			setLuzOverride(cmd.ID, 5*time.Second)
		}

		enviarComandoAtuador(cmd.ID, cmd.State, "cliente")
	}

	fmt.Printf("[GATEWAY] Cliente desconectado: %s\n", conn.RemoteAddr())
}

// ─── Broadcast / Estado ───────────────────────────────────────────────────────

func broadcastToClients(data []byte) {
	clientMux.Lock()
	defer clientMux.Unlock()
	for conn := range clients {
		_, _ = conn.Write(data)
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

// ─── Override de luz (cliente) ────────────────────────────────────────────────

func setLuzOverride(atuadorID string, d time.Duration) {
	luzOverrideMutex.Lock()
	luzOverrideAte[atuadorID] = time.Now().Add(d)
	luzOverrideMutex.Unlock()
}

func luzEmOverride(atuadorID string) bool {
	luzOverrideMutex.Lock()
	defer luzOverrideMutex.Unlock()

	t, ok := luzOverrideAte[atuadorID]
	if !ok {
		return false
	}
	if time.Now().After(t) {
		delete(luzOverrideAte, atuadorID)
		return false
	}
	return true
}
