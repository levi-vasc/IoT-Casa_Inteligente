package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// db é a instância global do banco de dados SQLite.
var db *sql.DB

// initDB abre (ou cria) o arquivo SQLite em dbPath e garante que a tabela
// de leituras dos sensores exista.
//
// O arquivo é criado automaticamente se não existir. Para usar um caminho
// customizado basta passar a variável de ambiente DB_PATH antes de iniciar
// o gateway (o padrão é "sensor_readings.db" no diretório de trabalho).
func initDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir banco de dados %s: %w", dbPath, err)
	}

	// Garante que a tabela existe.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sensor_readings (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			sensor_id TEXT    NOT NULL,
			type      TEXT    NOT NULL,
			value     REAL    NOT NULL,
			ts        DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sensor_ts ON sensor_readings (sensor_id, ts);
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela sensor_readings: %w", err)
	}

	return nil
}

// salvarLeitura persiste uma leitura de sensor no banco de dados.
// Apenas valores numéricos (float64) são salvos; leituras não-numéricas
// são ignoradas silenciosamente.
func salvarLeitura(data DeviceData) {
	val, ok := data.Value.(float64)
	if !ok {
		return
	}
	_, err := db.Exec(
		`INSERT INTO sensor_readings (sensor_id, type, value, ts) VALUES (?, ?, ?, ?)`,
		data.ID, data.Type, val, time.Now().UTC(),
	)
	if err != nil {
		fmt.Printf("[STORE] Erro ao salvar leitura de %s: %v\n", data.ID, err)
	}
}

// Leitura representa um ponto de série temporal retornado pela API de histórico.
type Leitura struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Type      string    `json:"type"`
}

// buscarHistorico retorna as leituras de um sensor entre from e to, ordenadas
// cronologicamente.
func buscarHistorico(sensorID string, from, to time.Time) ([]Leitura, error) {
	rows, err := db.Query(
		`SELECT ts, value, type
		   FROM sensor_readings
		  WHERE sensor_id = ? AND ts >= ? AND ts <= ?
		  ORDER BY ts ASC`,
		sensorID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar histórico: %w", err)
	}
	defer rows.Close()

	var leituras []Leitura
	for rows.Next() {
		var l Leitura
		if err := rows.Scan(&l.Timestamp, &l.Value, &l.Type); err != nil {
			return nil, fmt.Errorf("erro ao ler linha: %w", err)
		}
		leituras = append(leituras, l)
	}
	return leituras, rows.Err()
}
