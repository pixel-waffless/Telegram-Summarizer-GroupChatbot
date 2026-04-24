package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// MessageMember represents a message member
type MessageMember struct {
	Name    string
	Message string
}

// Table
func createTable(db *sql.DB) error {

	// Create table messages if not exists
	createTableMessage := `
	CREATE TABLE IF NOT EXISTS table_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		chat_id INTEGER NOT NULL,
		mensaje TEXT NOT NULL
	);`
	if _, err := db.Exec(createTableMessage); err != nil {
		log.Fatalf("Error creando tabla: %v", err)
	}
	// Create table prompt base if not exists
	createTablePrompt := `
	CREATE TABLE IF NOT EXISTS table_prompt_Base (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		prompt TEXT NOT NULL
	);`
	if _, err := db.Exec(createTablePrompt); err != nil {
		log.Fatalf("Error creando tabla: %v", err)
	}
	// Create table summaries if not exists
	createTableSummaries := `
	CREATE TABLE IF NOT EXISTS table_summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		summary TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTableSummaries); err != nil {
		log.Fatalf("Error creando tabla summaries: %v", err)
	}
	// Create table context chain if not exists
	createTableContextChain := `
	CREATE TABLE IF NOT EXISTS table_context_chain (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		compressed_context TEXT NOT NULL,
		token_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTableContextChain); err != nil {
		log.Fatalf("Error creando tabla context chain: %v", err)
	}
	return nil
}

// Prompt base insert

func insertPromptBase(db *sql.DB, chatID int64, prompt string) error {
	deleteQuery := `DELETE FROM table_prompt_Base WHERE chat_id = ?`
	if _, err := db.Exec(deleteQuery, chatID); err != nil {
		log.Printf("Error al eliminar prompt base anterior: %v", err)
	}
	query := `INSERT INTO table_prompt_Base(prompt,chat_id) VALUES (?,?)`
	_, err := db.Exec(query, prompt, chatID)
	return err
}

// Get prompt base
func getPromptBase(db *sql.DB, chatID int64) (string, error) {
	query := `SELECT prompt FROM table_prompt_Base WHERE chat_id = ? ORDER BY id DESC LIMIT 1`
	var prompt string
	err := db.QueryRow(query, chatID).Scan(&prompt)
	if err != nil {
		return "", err
	}
	return prompt, nil
}

func insertSummary(db *sql.DB, chatID int64, summary string) error {
	query := `INSERT INTO table_summaries (chat_id, summary) VALUES (?, ?)`
	_, err := db.Exec(query, chatID, summary)
	return err
}

func getLastSummary(db *sql.DB, chatID int64) (string, error) {
	query := `SELECT summary FROM table_summaries WHERE chat_id = ? ORDER BY id DESC LIMIT 1`
	var summary string
	err := db.QueryRow(query, chatID).Scan(&summary)
	if err != nil {
		return "", err
	}
	return summary, nil
}

func insertMessage(db *sql.DB, chatID int64, m MessageMember) error {
	query := `INSERT INTO table_messages (name, chat_id, mensaje) VALUES (?, ?, ?)`
	_, err := db.Exec(query, m.Name, chatID, m.Message)
	return err
}

// GetAll returns the last N messages saved in chronological order for a specific chat (if limit <=0, returns all)
func GetAll(db *sql.DB, chatID int64, limit int) []MessageMember {
	var query string
	var rows *sql.Rows
	var err error
	if limit > 0 {
		query = `SELECT name, mensaje FROM table_messages WHERE chat_id = ? ORDER BY id DESC LIMIT ?`
		rows, err = db.Query(query, chatID, limit)
	} else {
		query = `SELECT name, mensaje FROM table_messages WHERE chat_id = ? ORDER BY id ASC`
		rows, err = db.Query(query, chatID)
	}
	if err != nil {
		log.Printf("Error al obtener mensajes: %v", err)
		return nil
	}
	defer rows.Close()

	var messages []MessageMember
	for rows.Next() {
		var m MessageMember
		if err := rows.Scan(&m.Name, &m.Message); err != nil {
			log.Printf("Error al escanear mensaje: %v", err)
			continue
		}
		messages = append(messages, m)
	}
	if limit > 0 {
		// Revertir el orden para que sea cronológico
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}
	return messages
}

// GetFormattedMessages returns the messages as formatted string for a specific chat, limiting to the last 300
func GetFormattedMessages(db *sql.DB, chatID int64, limit int) string {
	messages := GetAll(db, chatID, limit)
	if len(messages) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, msg := range messages {
		builder.WriteString(msg.Message)
		builder.WriteString("\n")
	}
	return builder.String()
}

// GetStats returns statistics of the buffer (messages)
func GetStats(db *sql.DB, chatID int64) (string, error) {
	query := "SELECT COUNT(*) FROM table_messages WHERE chat_id = ?;"
	var count int
	err := db.QueryRow(query, chatID).Scan(&count)
	if err != nil {
		log.Printf("Error scanning message: %v", err)
		return "Could not retrieve data", nil
	}

	return fmt.Sprintf("📊 *Message Statistics:*\n- Saved messages: %d/%d\n", count, 300), nil
}

// Clear table messages for a specific chat
func Clear(db *sql.DB, chatID int64) {
	query := `DELETE FROM table_messages WHERE chat_id = ?`
	db.Exec(query, chatID)
}

// estimateTokens estimates approximately the tokens of a string (1 token ≈ 4 characters)
func estimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// insertCompressedContext saves the compressed context of a chat
func insertCompressedContext(db *sql.DB, chatID int64, context string) error {
	tokenCount := estimateTokens(context)
	query := `INSERT INTO table_context_chain (chat_id, compressed_context, token_count) VALUES (?, ?, ?)`
	_, err := db.Exec(query, chatID, context, tokenCount)
	return err
}

// getCompressedContext gets the current compressed context of a chat
func getCompressedContext(db *sql.DB, chatID int64) (string, error) {
	query := `SELECT compressed_context FROM table_context_chain WHERE chat_id = ? ORDER BY id DESC LIMIT 1`
	var context string
	err := db.QueryRow(query, chatID).Scan(&context)
	if err != nil {
		return "", err
	}
	return context, nil
}

// getTotalTokens calculates the total tokens of compressed context + unprocessed messages
func getTotalTokens(db *sql.DB, chatID int64) int {
	// Get compressed context
	compressedContext, _ := getCompressedContext(db, chatID)
	tokens := estimateTokens(compressedContext)

	// Get all unprocessed messages
	messages := GetAll(db, chatID, 0)
	for _, msg := range messages {
		tokens += estimateTokens(msg.Name + ": " + msg.Message)
	}

	return tokens
}

// shouldCompress checks if context compression is necessary (when reaching ~100k tokens)
func shouldCompress(db *sql.DB, chatID int64) bool {
	tokens := getTotalTokens(db, chatID)
	return tokens > 100000
}
