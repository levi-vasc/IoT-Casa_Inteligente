# IoT — Rota das Coisas

Sistema IoT em Go com gateway central, sensores simulados, atuadores e cliente interativo.

## Arquitetura

| Componente | Porta | Protocolo | Função |
|---|---|---|---|
| Gateway | `:8080` | TCP | Comandos e broadcast para clientes |
| Gateway | `:8081` | UDP | Ingestão de dados dos sensores |
| Gateway | `:8082` | HTTP | API de histórico (séries temporais) |
| Gateway | `:9000` | TCP | Controle de atuadores |
| Gateway | `:9001` | TCP | Consulta de estado por sensores |

## Executar com Docker Compose

```bash
docker-compose up --build
```

## Persistência SQLite

O gateway persiste cada leitura de sensor em um arquivo **SQLite** criado automaticamente no diretório de trabalho.

- **Arquivo padrão:** `sensor_readings.db`
- **Variável de ambiente:** `DB_PATH` — define o caminho do arquivo SQLite
  ```
  DB_PATH=/data/sensor_readings.db
  ```
- A tabela `sensor_readings` é criada automaticamente na primeira execução.

Schema:
```sql
CREATE TABLE sensor_readings (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    sensor_id TEXT    NOT NULL,
    type      TEXT    NOT NULL,
    value     REAL    NOT NULL,
    ts        DATETIME NOT NULL
);
```

## API de Histórico

### `GET /sensors/{id}/history`

Retorna as leituras de um sensor em um intervalo de tempo, ordenadas cronologicamente.

**Parâmetros de query (obrigatórios):**
- `from` — início do intervalo (formato RFC3339)
- `to`   — fim do intervalo (formato RFC3339)

**Exemplo de requisição:**
```
GET http://localhost:8082/sensors/temp01/history?from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z
```

**Exemplo de resposta:**
```json
{
  "sensor_id": "temp01",
  "from": "2024-01-01T00:00:00Z",
  "to": "2024-12-31T23:59:59Z",
  "readings": [
    { "timestamp": "2024-06-15T10:30:00Z", "value": 24.5, "type": "temperatura" },
    { "timestamp": "2024-06-15T10:30:01Z", "value": 25.1, "type": "temperatura" }
  ]
}
```

**Códigos de resposta:**
- `200 OK` — leituras retornadas com sucesso
- `400 Bad Request` — parâmetros ausentes ou formato inválido
- `500 Internal Server Error` — erro ao consultar o banco de dados
