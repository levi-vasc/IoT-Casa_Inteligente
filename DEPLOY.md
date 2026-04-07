# Guia de Deployment — IoT Casa Inteligente

## Arquitetura

```
PC Linux (Host):
  └─ Gateway (executável Go nativo)
       ├─ :8080 TCP ← Clientes
       ├─ :8081 UDP ← Sensores
       └─ :9000 TCP ← Atuadores

Docker (mesmo PC ou PC diferente na LAN):
  ├─ temp01, temp02   (sensores de temperatura → UDP :8081)
  ├─ pres01, pres02   (sensores de presença    → UDP :8081)
  ├─ ar01,   ar02     (atuadores AC            → TCP :9000)
  ├─ luz01,  luz02    (atuadores de luz        → TCP :9000)
  └─ client           (cliente interativo      → TCP :8080)
```

O gateway escuta em `0.0.0.0`, ou seja, aceita conexões de qualquer interface de rede do host, incluindo a bridge do Docker (`docker0`, padrão `172.17.0.1`).

---

## Pré-requisitos

| Componente | Versão mínima |
|------------|--------------|
| Go         | 1.21         |
| Docker     | 20.10        |
| Docker Compose | v2      |

---

## 1. Executar o Gateway no PC (host nativo)

### 1.1 Compilar e executar com o script

```bash
chmod +x scripts/run-gateway.sh
./scripts/run-gateway.sh
```

### 1.2 Compilar e executar manualmente

```bash
cd gateway
go mod init gateway   # somente na primeira vez
go build -o gateway .
./gateway
```

O gateway vai imprimir:

```
[GATEWAY] Sistema iniciado com sucesso
[GATEWAY] Sensores (UDP)  -> :8081
[GATEWAY] Clientes (TCP)  -> :8080
[GATEWAY] Atuadores (TCP) -> :9000
```

---

## 2. Descobrir o IP do host para os containers Docker

### Opção A — IP da bridge docker0 (padrão Linux)

```bash
ip addr show docker0 | grep "inet " | awk '{print $2}' | cut -d/ -f1
# Resultado típico: 172.17.0.1
```

### Opção B — IP na rede LAN (quando gateway e Docker estão em PCs diferentes)

```bash
ip route get 8.8.8.8 | awk '{print $7; exit}'
# Ou
hostname -I | awk '{print $1}'
```

---

## 3. Configurar e executar os containers Docker

### 3.1 Criar o arquivo `.env`

```bash
cp .env.example .env
```

Edite `.env` com o IP correto do host onde o gateway está rodando:

```dotenv
# Gateway no mesmo PC (Linux) — usar IP da bridge docker0
GATEWAY_HOST=172.17.0.1

# Gateway em outro PC da LAN — usar IP real do PC
# GATEWAY_HOST=192.168.1.100
```

### 3.2 Subir os containers

```bash
docker compose up -d
```

### 3.3 Acompanhar logs

```bash
docker compose logs -f
```

### 3.4 Abrir o cliente interativo

```bash
docker attach client
```

---

## 4. Variáveis de Ambiente

### docker-compose / `.env`

| Variável       | Padrão        | Descrição                                      |
|----------------|---------------|------------------------------------------------|
| `GATEWAY_HOST` | `172.17.0.1`  | IP do host onde o gateway está executando       |

### Serviços individuais (definidas no docker-compose.yml)

| Variável       | Exemplo                    | Serviço         |
|----------------|---------------------------|-----------------|
| `GATEWAY_ADDR` | `172.17.0.1:8081`         | sensores (UDP)  |
| `GATEWAY_ADDR` | `172.17.0.1:9000`         | atuadores (TCP) |
| `GATEWAY_ADDR` | `172.17.0.1:8080`         | cliente (TCP)   |
| `SENSOR_ID`    | `temp01`                  | sensores        |
| `ATUADOR_ID`   | `ar01`                    | atuadores       |

---

## 5. Cenários de Deployment

### Cenário 1: Gateway e Docker no mesmo PC Linux

```
GATEWAY_HOST=172.17.0.1
```

1. Inicie o gateway: `./scripts/run-gateway.sh`
2. Suba os containers: `docker compose up -d`

### Cenário 2: Gateway em PC A, Docker em PC B (mesma LAN)

No PC B (Docker):

```dotenv
# .env
GATEWAY_HOST=192.168.1.50   # IP do PC A na LAN
```

No PC A:

```bash
./scripts/run-gateway.sh
```

No PC B:

```bash
docker compose up -d
```

> **Atenção:** Certifique-se de que o firewall do PC A (gateway) permite conexões nas portas 8080/TCP, 8081/UDP e 9000/TCP.

---

## 6. Liberar portas no firewall (Linux)

```bash
# UFW
sudo ufw allow 8080/tcp
sudo ufw allow 8081/udp
sudo ufw allow 9000/tcp

# iptables (alternativa)
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
sudo iptables -A INPUT -p udp --dport 8081 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 9000 -j ACCEPT
```

---

## 7. Encerrar o sistema

```bash
# Parar containers
docker compose down

# Parar o gateway (processo em foreground)
# Pressione Ctrl+C no terminal onde o gateway está rodando
```
