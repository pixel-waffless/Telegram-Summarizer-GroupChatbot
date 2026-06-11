package main

import (
	"database/sql"
	"errors"
	"log"
)

// autoCompressContext verifica y comprime automáticamente el contexto si es necesario
func CompressContext(db *sql.DB, chatID int64) (bool, error) {
	if !shouldCompress(db, chatID) {
		return false, nil // No es necesario comprimir
	}

	log.Printf("Comprimiendo contexto para chat %d", chatID)

	// Obtener el contexto comprimido anterior
	previousContext, _ := getCompressedContext(db, chatID)

	// Obtener los mensajes sin procesar
	messages := GetAll(db, chatID, 0)
	if len(messages) == 0 && previousContext == "" {
		return false, nil // No hay nada que comprimir
	}

	// Construir el texto a resumir
	var textToCompress string
	if previousContext != "" {
		textToCompress = "Contexto anterior resumido:\n" + previousContext + "\n\nNuevos mensajes:\n"
	}

	for _, msg := range messages {
		textToCompress += msg.Name + ": " + msg.Message + "\n"
	}

	// Intentar con GEMINI
	compressedText, err := waifuSummaryGEMINI(textToCompress, promptContextCompress)

	if err != nil {
		log.Printf("Error con GEMINI al comprimir: %v", err)
	}

	// Si falla, intentar con GROQ
	if compressedText == "" {
		compressedText, err = groqIA(textToCompress, promptContextCompress)
		if err != nil {
			log.Printf("Error con GROQ al comprimir: %v", err)
			return false, nil
		}
	}

	if compressedText == "" {
		return false, nil
	}

	// Guardar el nuevo contexto comprimido
	if err := insertCompressedContext(db, chatID, compressedText); err != nil {
		log.Printf("Error al guardar contexto comprimido: %v", err)
		return false, errors.New("failed to save compressed context")
	}

	// Limpiar los mensajes sin procesar
	Clear(db, chatID)

	log.Printf("Contexto comprimido exitosamente. Nuevos tokens estimados: %d", estimateTokens(compressedText))

	return true, nil
}
