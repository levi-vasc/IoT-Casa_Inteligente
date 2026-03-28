package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// ===== SENSOR =====
type SensorData struct {
	ID    string  `json:"id"`
	Tipo  string  `json:"type"`
	Valor float64 `json:"value"`
}

// ===== ATUADOR =====
type Atuador struct {
	ID     string `json:"id"`
	Tipo   string `json:"type"`
	Estado bool   `json:"state"`
	Auto   bool   `json:"auto"`
}

// ===== ARMAZENAMENTO =====
var sensors = make(map[string]SensorData)
var mutex = &sync.Mutex{}

var atuadores = map[string]Atuador{
	"ar01":    {ID: "ar01", Tipo: "ar", Estado: false, Auto: true},
	"luz01":   {ID: "luz01", Tipo: "luz", Estado: false, Auto: true},
	"porta01": {ID: "porta01", Tipo: "porta", Estado: false, Auto: true},
}

// ===== CONEXÕES COM ATUADORES =====
var connsAtuadores = map[string]net.Conn{}

func conectarAtuador(id, addr string) {
	for {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			fmt.Printf("[GATEWAY] Aguardando atuador %s...\n", id)
			time.Sleep(2 * time.Second)
			continue
		}
		mutex.Lock()
		connsAtuadores[id] = conn
		mutex.Unlock()
		fmt.Printf("[GATEWAY] Conectado ao atuador %s\n", id)
		return
	}
}

func conectarAtuadores() {
	enderecos := map[string]string{
		"ar01":    "localhost:9001",
		"luz01":   "localhost:9002",
		"porta01": "localhost:9003",
	}

	var wg sync.WaitGroup
	for id, addr := range enderecos {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()
			conectarAtuador(id, addr)
		}(id, addr)
	}
	wg.Wait()
}

func enviarComandoAtuador(a Atuador) {
	mutex.Lock()
	conn, ok := connsAtuadores[a.ID]
	mutex.Unlock()

	if !ok {
		return
	}

	dados, _ := json.Marshal(a)
	dados = append(dados, '\n')

	_, err := conn.Write(dados)
	if err != nil {
		fmt.Printf("[ERRO] Conexão perdida com %s, reconectando...\n", a.ID)
		mutex.Lock()
		delete(connsAtuadores, a.ID)
		mutex.Unlock()

		enderecos := map[string]string{
			"ar01":    "localhost:9001",
			"luz01":   "localhost:9002",
			"porta01": "localhost:9003",
		}
		go conectarAtuador(a.ID, enderecos[a.ID])
	}
}

// ================= UDP =================
func servidorUDP() {
	addr, _ := net.ResolveUDPAddr("udp", ":8081")
	conn, _ := net.ListenUDP("udp", addr)

	fmt.Println("[GATEWAY] UDP na porta 8081...")

	buffer := make([]byte, 1024)

	for {
		n, _, _ := conn.ReadFromUDP(buffer)

		var data SensorData
		if err := json.Unmarshal(buffer[:n], &data); err != nil {
			continue
		}

		mutex.Lock()
		sensors[data.ID] = data
		mutex.Unlock()

		fmt.Printf("[SENSOR] %s | tipo: %-12s | valor: %.2f\n", data.ID, data.Tipo, data.Valor)

		controlarAtuadores(data)
	}
}

// ===== CONTROLE AUTOMÁTICO =====
func controlarAtuadores(data SensorData) {

	// ===== AR-CONDICIONADO =====
	if data.Tipo == "temperatura" {
		ar := atuadores["ar01"]
		if ar.Auto {
			ar.Estado = data.Valor > 25
			atuadores["ar01"] = ar
			enviarComandoAtuador(ar)
		}
	}

	// ===== LÂMPADA =====
	if data.Tipo == "luminosidade" {
		luz := atuadores["luz01"]
		if luz.Auto {
			luz.Estado = data.Valor < 300
			atuadores["luz01"] = luz
			enviarComandoAtuador(luz)
		}
	}
}

// ================= TCP =================
func servidorTCP() {
	listener, _ := net.Listen("tcp", ":8080")

	fmt.Println("[GATEWAY] TCP na porta 8080...")

	for {
		conn, _ := listener.Accept()
		go handleCliente(conn)
	}
}

// ===== CLIENTE =====
func handleCliente(conn net.Conn) {
	defer conn.Close()

	fmt.Println("[GATEWAY] Cliente conectado:", conn.RemoteAddr())

	go enviarDados(conn)

	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			return
		}

		var cmd Atuador
		if err = json.Unmarshal(buffer[:n], &cmd); err != nil {
			continue
		}

		atuador := atuadores[cmd.ID]
		atuador.Estado = cmd.Estado
		atuador.Auto = cmd.Auto
		atuadores[cmd.ID] = atuador

		enviarComandoAtuador(atuador)

		fmt.Printf("[COMANDO] %s | estado: %v | auto: %v\n", atuador.ID, atuador.Estado, atuador.Auto)
	}
}

// ===== ENVIO DE DADOS =====
func enviarDados(conn net.Conn) {
	for {
		mutex.Lock()
		for _, s := range sensors {
			jsonData, _ := json.Marshal(s)
			conn.Write(jsonData)
			conn.Write([]byte("\n"))
		}
		mutex.Unlock()

		for _, a := range atuadores {
			jsonData, _ := json.Marshal(a)
			conn.Write(jsonData)
			conn.Write([]byte("\n"))
		}

		time.Sleep(1 * time.Second)
	}
}
