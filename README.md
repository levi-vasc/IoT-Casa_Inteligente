# IoT-Casa_Inteligente

Este projeto é a primeira etapa do problema de MI de Concorrência e Conectividade, semestre 2026.1, da Universidade Estadual de Feira de Santana (UEFS). Ele consiste na simulação de uma casa inteligente, na qual cada entidade (sensores, atuadores e demais componentes) atua de maneira independente, comunicando-se por meio de mecanismos de conectividade definidos pela aplicação. O objetivo é exercitar conceitos de concorrência, comunicação e coordenação entre componentes autônomos em um ambiente de IoT.

## Índice
- [Descrição do Projeto](#descrição-do-projeto)
- [Arquitetura](# arquitetura)
- [Estrutura de Comunicação e Mensagens](#estrutura-de-comunicação-e-mensagem)
- [Configuração do Ambiente](#configuração-do-ambiente)
- [Modo de Uso](#modo-de-uso)
- [Testes](#testes)

---

## 📝 Descrição do Projeto

O **IoT Casa Inteligente** é um projeto desenvolvido em **Go** (com suporte via **Docker**) para construir uma solução de automação residencial baseada em Internet das Coisas (IoT). O objetivo é integrar e controlar dispositivos e sensores de uma casa (iluminação, presença, temperatura, ar condicionado) por meio de uma aplicação central, permitindo monitoramento em **tempo real**, **automatizações** e **acionamentos remotos** de forma organizada e escalável.

Na simulação, cada entidade atua de maneira independente, com seu próprio ciclo de execução e responsabilidades, refletindo o comportamento típico de dispositivos IoT no mundo real. Essa independência permite exercitar, de forma prática, os principais desafios do domínio: concorrência (várias entidades executando simultaneamente), conectividade (troca de mensagens/dados entre componentes) e coordenação (garantir consistência e respostas corretas mesmo com eventos ocorrendo em paralelo).

---

## 🏗️ Arquitetura

A simulação segue um modelo de casa inteligente com gateway central: as **entidades** (sensores e atuadores) executam de forma independente e concorrente, enquanto o **servidor** atua como controlador e ponto de integração, recebendo mensagens via rede (**TCP/UDP**), mantendo o estado do ambiente e coordenando comandos/respostas.

### Componentes

  * Dispositivos
      * Sensores: produzem eventos e enviam ao gateway (presença e temperatura)
      * Atuadores: recebem comandos do gateway e alteram seu estado (lâmpadas e ares-condicionados)
      * Cada dispositivo roda em seu próprio fluxo de execução (goroutine) e envia dados em um intervalo de **1 segundo**
  * Gateway/Servidor
      * Recebe e processa mensagens dos dispositivos via TCP/UDP
      * Mantém um estado consolidado (últimas leituras, estado de atuadores, disponibilidade/conexão)
      * Aplica regras e envia comandos aos atuadores
      * Funciona como a "ponte" entre conectividade e lógica do sistema
  * Cliente
      * Interface do usuário para visualizar sensores e atuadores e seus estados atuais
      * Permite ligar/desligar atuadores, enviando comandos ao gateway
      * Se comunica com o gateway via **TCP** e pode também receber/consultar atualizações

### Visão geral

<p align="center">
 <img width="574" height="262" alt="dg_redes" src="https://github.com/user-attachments/assets/58020a1d-a759-44c6-946c-79a89c0c2649" />
</p>

#### 1. Sensores

| SENSOR_ID | Função | Valores |
| --- | --- | --- |
| temp01 | Envia dados da temperatura do quarto 1 para o servidor | 18 a 35° |
| temp02 | Envia dados da temperatura do quarto 2 para o servidor | 18 a 35° |
| pres01 | Detecta presença no quarto 1 e envia valor para o servidor | 0 ou 1 |
| pres02 | Detecta presença no quarto 2 e envia valor para o servidor | 0 ou 1 |

>[!NOTE]
> Os sensores de temperatura geram leituras com **baixa variância**, variando lentamente ao longo do tempo. Quando o ar condicionado é ligado, os valores decrescem gradativamente. Para isso, eles recebem a informação de estado do atuador através da porta `:8080`

#### 2. Atuadores

| ATUADOR_ID | Função | Condição LIGADO | Condição DESLIGADO |
| --- | --- | --- | --- |
| ar01 | Resfriar quarto 1 quando necessário | temp01 >= 26° | temp01 <= 20° |
| ar02 | Resfriar quarto 2 quando necessário | temp02 >= 26° | temp02 <= 20° |
| luz01 | Lâmpada do quarto 1 ascende | pres01 = 1 | pres01 = 0 |
| luz02 | Lâmpada do quarto 2 ascende | pres02 = 1 | pres02 = 0 |

#### 3. Gateway

| Porta | Tipo | Propósito |
| --- | --- | --- |
| 8081 | UDP | Recebimento de dados dos sensores, envio do estado do AC para sensores de temperatura |
| 8080 | TCP | Comunicação com o cliente |
| 9000 | TCP | Envio/recebimento de dados para os atuadores | 

#### 4. Cliente

O cliente é a entidade responsável por **monitorar** e **interagir** com a casa inteligente. Ele consulta o **gateway** para exibir, no terminal, a lista de sensores/atuadores e seus estados atuais, e permite ligar/desligar atuadores manualmente, enviando comandos ao servidor. No sistema, mais de um cliente pode atuar simulataneamente.

---

## 📨 Estrutura de Comunicação e Mensagens

Para garantir que o gateway compreenda as informações vindas de diferentes dispositivos, utilizamos um protocolo de mensagens padronizado em **JSON** via sockets.

* Formato dos pacotes (sensores para gateway)

```json
{
  "id": "temp01",
  "tipo": "temperatura",
  "valor": 24.5
}
```

* Formato dos comandos (gateway para atuadores)

```json
{
  "id": "luz01",
  "type": "comando",
  "state": true
}
```

* Formato da notificação (estado do atuador)

```json
{
  "id": "ar01",
  "type": "estado",
  "state": true
}
```
---

 ## ⚙️ Configuração do Ambiente

 ### Requisitos
  * Go instalado (Versão utilizada: 1.22)
  * Docker e Docker Compose para executar em contêiner

 ### Clonar o repositório

 ```
 git clone https://github.com/levi-vasc/IoT-Casa_Inteligente.git
cd IoT-Casa_Inteligente
 ```

### Parâmetros de rede
 * Host do gateway (`GATEWAY_IP`): IP da máquina do servidor
 * Porta sensores: 8081 - UDP
 * Porta atuadores: 9000 - TCP
 * Porta cliente: 8080 - TCP

### Arquivo .env

Na raíz do projeto, há um arquivo `.env` que define a variável `GATEWAY_IP`. Essa variável deve ser alterada para o **IP da máquina do servidor**.

```
# Exemplo
GATEWAY_IP=172.16.103.10
```

---

## ▶️ Modo de Uso

> [!IMPORTANT]
> Para rodar o sistema em máquinas diferentes, certifique-se de que ambas estejam na mesma rede local e que o firewall permita conexões nas portas utilizadas.

### 1) Configuração (Caso use máquinas diferentes)

Antes de iniciar, informe aos dispositivos o endereço IP da máquina que executará o **Gateway**. No arquivo `.env` da máquina dos dispositivos, altere:

```
# Exemplo
GATEWAY_IP=172.16.103.10
```

### 2) Executar servidor (Máquina A)

Em um terminal, no diretório do projeto:

```
docker compose up -d gateway
```

### 3) Executar dispositivos (Máquina B)

No terminal de outro computador, inicie os dispositivos:

```
docker compose up -d
docker compose run client
```

### 4) Menu do cliente

Ao executar o cliente, aparecerá no terminal um menu com opções:

```
Casa Inteligente

1. Visualizar sensores
2. Visualizar atuadores
3. Ligar/Desligar atuadores
0. Sair
```

Escolhendo a opção 1, é possível visualizar os dados enviados por `temp01`, `temp02`, `pres01` e `pres02` em tempo real. Um exemplo:

```
Sensores - Tempo Real

temp01 22.3º
temp02 27.5º

pres01 Presente
pres02 Ausente
```

Na opção 2, podemos ver o estado dos atuadores:

```
Atuadores

ar01 DESLIGADO
ar02 LIGADO

luz01 LIGADA
luz02 DESLIGADA
```

Ainda há a opção 3, que é interativa. É nela que o cliente pode desligar ou ligar atuadores manualmente:

```
Controle de usuário

ar01 DESLIGADO
ar02 LIGADO

luz01 LIGADA
luz02 DESLIGADA

Digite o ID do atuador:
(1 - Ligar, 2- Desligar):
```

---

## 🧪 Testes

Durante os testes, foi possível **visualizar os dados** de sensores e atuadores e **controlar manualmente** o estado dos atuadores, com atualização em tempo real.

O principal problema observado está relacionado à **reconexão do cliente com o gateway**: se o gateway cair, o cliente detecta a desconexão (mensagem exibida no terminal), porém **não volta a receber atualizações** quando o gateway é iniciado novamente. Após a queda, os dados permanecem congelados mesmo com o servidor operando normalmente.

### 1. Tela inicial com o servidor conectado

<p align="center">
 <img width="311" height="285" alt="image" src="https://github.com/user-attachments/assets/a3eb2831-d036-4de1-a38a-f67f18718693" />
</p> 

### 2. Desconectando o servidor

```
docker compose down gateway
```

### 3. Tela inicial com o servidor desconectado

<p align="center">
 <img width="313" height="277" alt="image" src="https://github.com/user-attachments/assets/54e9ac3b-0850-412f-8636-fb7d3333fdf5" />
</p> 

### 4. Reconectando o servidor

```
docker compose up gateway
``` 

Mesmo após a reconexão, o cliente ainda mostra o servidor como desconectado, e os dados continuam congelados. Isso é um **ponto de melhoria futuro** (ex.: implementar estratégia de reconexão/backoff e re-sincronização de estado).
