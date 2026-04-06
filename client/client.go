package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// ── ANSI ──────────────────────────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	clr    = "\033[H\033[2J"
)

// ── Estado ────────────────────────────────────────────────────────────────────

type Estado struct {
	mu        sync.RWMutex
	temp      map[string]float64
	presenca  map[string]bool
	atuadores map[string]bool
	conectado bool
	updates   chan struct{}
}

var e = &Estado{
	temp:      map[string]float64{"temp01": 0, "temp02": 0},
	presenca:  map[string]bool{"pres01": false, "pres02": false},
	atuadores: map[string]bool{"ar01": false, "ar02": false, "luz01": false, "luz02": false},
	updates:   make(chan struct{}, 1),
}

func notificar() {
	select {
	case e.updates <- struct{}{}:
	default:
	}
}

// ── Recepção de mensagens do gateway ─────────────────────────────────────────

type Msg struct {
	ID    string      `json:"id"`
	Type  string      `json:"type,omitempty"`
	Value interface{} `json:"value,omitempty"`
	State bool        `json:"state"`
}

func receberMensagens(conn net.Conn) {
	decoder := json.NewDecoder(conn)
	for {
		var msg Msg
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		e.mu.Lock()
		switch msg.Type {
		case "temperatura":
			if v, ok := msg.Value.(float64); ok {
				e.temp[msg.ID] = v
			}
		case "presenca":
			if v, ok := msg.Value.(float64); ok {
				e.presenca[msg.ID] = v == 1
			}
		default:
			if _, ok := e.atuadores[msg.ID]; ok {
				e.atuadores[msg.ID] = msg.State
			}
		}
		e.mu.Unlock()
		notificar()
	}
	e.mu.Lock()
	e.conectado = false
	e.mu.Unlock()
	notificar()
}

// ── Helpers de tela ───────────────────────────────────────────────────────────

func cabecalho(titulo string) {
	fmt.Print(clr)
	fmt.Printf("%s╔══ CASA INTELIGENTE ══╗%s\n", bold+cyan, reset)
	fmt.Printf("  %s%s%s\n\n", bold, titulo, reset)
}

func statusConexao() string {
	e.mu.RLock()
	c := e.conectado
	e.mu.RUnlock()
	if c {
		return green + "● Conectado" + reset
	}
	return red + "● Desconectado" + reset
}

// aguardarEnter lança uma goroutine que lê stdin e avisa pelo canal retornado.
func aguardarEnter() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		bufio.NewReader(os.Stdin).ReadString('\n')
		close(ch)
	}()
	return ch
}

// ── Tela: Menu ────────────────────────────────────────────────────────────────

func telaMenu() string {
	cabecalho("MENU PRINCIPAL")
	fmt.Printf("  Servidor %s\n\n", statusConexao())
	fmt.Printf("  %s1.%s Visualizar sensores\n", yellow, reset)
	fmt.Printf("  %s2.%s Visualizar atuadores\n", yellow, reset)
	fmt.Printf("  %s3.%s Ligar / Desligar atuador\n", yellow, reset)
	fmt.Printf("  %s0.%s Sair\n", dim, reset)
	fmt.Printf("\n  Opcao: ")
	opt, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(opt)
}

// ── Tela: Sensores (tempo real) ───────────────────────────────────────────────

func renderSensores() {
	cabecalho("SENSORES — Tempo Real")
	fmt.Printf("  Servidor %s\n\n", statusConexao())

	e.mu.RLock()
	defer e.mu.RUnlock()

	fmt.Printf("  %sTEMPERATURA%s\n", dim, reset)
	for _, id := range []string{"temp01", "temp02"} {
		v := e.temp[id]
		cor := cyan
		if v > 30 {
			cor = red
		} else if v > 27 {
			cor = yellow
		}
		fmt.Printf("    %-8s  %s%5.1f C%s\n", id, cor+bold, v, reset)
	}

	fmt.Printf("\n  %sPRESENCA%s\n", dim, reset)
	for _, id := range []string{"pres01", "pres02"} {
		if e.presenca[id] {
			fmt.Printf("    %-8s  %s● Presente%s\n", id, green, reset)
		} else {
			fmt.Printf("    %-8s  %s○ Ausente%s\n", id, dim, reset)
		}
	}

	fmt.Printf("\n  %sEnter para voltar...%s", dim, reset)
}

func telaSensores() {
	renderSensores()
	voltar := aguardarEnter()
	for {
		select {
		case <-voltar:
			return
		case <-e.updates:
			renderSensores()
		}
	}
}

// ── Tela: Atuadores (tempo real) ─────────────────────────────────────────────

func renderAtuadores() {
	cabecalho("ATUADORES — Tempo Real")
	fmt.Printf("  Servidor %s\n\n", statusConexao())

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, id := range []string{"ar01", "ar02", "luz01", "luz02"} {
		icone := "[AC]"
		if strings.HasPrefix(id, "luz") {
			icone = "[LZ]"
		}
		if e.atuadores[id] {
			fmt.Printf("    %s %s%-8s%s  %sLIGADO%s\n", icone, bold, id, reset, green, reset)
		} else {
			fmt.Printf("    %s %-8s  %sDESLIGADO%s\n", icone, id, dim, reset)
		}
	}

	fmt.Printf("\n  %sEnter para voltar...%s", dim, reset)
}

func telaAtuadores() {
	renderAtuadores()
	voltar := aguardarEnter()
	for {
		select {
		case <-voltar:
			return
		case <-e.updates:
			renderAtuadores()
		}
	}
}

// ── Tela: Controle ────────────────────────────────────────────────────────────

func enviarComandoAoGateway(gatewayAddr, id string, ligado bool) error {
	conn, err := net.Dial("tcp", gatewayAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	return enc.Encode(map[string]interface{}{"id": id, "state": ligado})
}

func telaControle(gatewayAddr string) {
	cabecalho("LIGAR / DESLIGAR ATUADOR")

	e.mu.RLock()
	fmt.Printf("  %sESTADO ATUAL%s\n", dim, reset)
	for _, id := range []string{"ar01", "ar02", "luz01", "luz02"} {
		icone := "[AC]"
		if strings.HasPrefix(id, "luz") {
			icone = "[LZ]"
		}
		if e.atuadores[id] {
			fmt.Printf("    %s %-8s  %sLIGADO%s\n", icone, id, green, reset)
		} else {
			fmt.Printf("    %s %-8s  %sDESLIGADO%s\n", icone, id, dim, reset)
		}
	}
	e.mu.RUnlock()

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n  Atuador (ar01, ar02, luz01, luz02): ")
	id, _ := reader.ReadString('\n')
	id = strings.TrimSpace(id)

	e.mu.RLock()
	_, idValido := e.atuadores[id]
	e.mu.RUnlock()

	if !idValido {
		fmt.Printf("  %s[ERRO] ID invalido.%s\n", red, reset)
		fmt.Printf("  %sEnter para voltar...%s ", dim, reset)
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	fmt.Printf("  Acao (1=Ligar, 2=Desligar): ")
	acao, _ := reader.ReadString('\n')
	acao = strings.TrimSpace(acao)

	var ligado bool
	switch acao {
	case "1":
		ligado = true
	case "2":
		ligado = false
	default:
		fmt.Printf("  %s[ERRO] Opcao invalida.%s\n", red, reset)
		fmt.Printf("  %sEnter para voltar...%s ", dim, reset)
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	if err := enviarComandoAoGateway(gatewayAddr, id, ligado); err != nil {
		fmt.Printf("  %s[ERRO] Falha ao enviar: %v%s\n", red, err, reset)
		fmt.Printf("  %sEnter para voltar...%s ", dim, reset)
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	estadoStr := green + "LIGADO" + reset
	if !ligado {
		estadoStr = red + "DESLIGADO" + reset
	}
	fmt.Printf("\n  Comando enviado: %s%s%s → %s\n", bold, id, reset, estadoStr)
	fmt.Printf("  %sEnter para voltar...%s ", dim, reset)
	bufio.NewReader(os.Stdin).ReadString('\n')
}

// ── Main ───────────────────────────────────────────────────────────────────────

func main() {
	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:8080"
	}

	fmt.Print(clr)
	fmt.Printf("%s╔══ CASA INTELIGENTE ══╗%s\n", bold+cyan, reset)
	fmt.Printf("  Conectando ao servidor %s...\n", gatewayAddr)

	conn, err := net.Dial("tcp", gatewayAddr)
	if err != nil {
		fmt.Printf("  %s[ERRO] %v%s\n", red, err, reset)
		os.Exit(1)
	}
	defer conn.Close()

	e.mu.Lock()
	e.conectado = true
	e.mu.Unlock()

	go receberMensagens(conn)

	for {
		switch telaMenu() {
		case "1":
			telaSensores()
		case "2":
			telaAtuadores()
		case "3":
			telaControle(gatewayAddr)
		case "0":
			fmt.Print(clr)
			fmt.Println("Saindo...")
			return
		}
	}
}
