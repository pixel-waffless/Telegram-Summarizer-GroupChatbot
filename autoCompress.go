package main

import (
	"database/sql"
	"log"
)

// autoCompressContext verifica y comprime automáticamente el contexto si es necesario
func autoCompressContext(db *sql.DB, chatID int64) error {
	if !shouldCompress(db, chatID) {
		return nil // No necesita compresión aún
	}

	log.Printf("Comprimiendo contexto para chat %d (tokens > 100k)", chatID)

	// Obtener el contexto comprimido anterior
	previousContext, _ := getCompressedContext(db, chatID)

	// Obtener los mensajes sin procesar
	messages := GetAll(db, chatID, 0)
	if len(messages) == 0 && previousContext == "" {
		return nil // No hay nada que comprimir
	}

	// Construir el texto a resumir
	var textToCompress string
	if previousContext != "" {
		textToCompress = "Contexto anterior resumido:\n" + previousContext + "\n\nNuevos mensajes:\n"
	}

	for _, msg := range messages {
		textToCompress += msg.Name + ": " + msg.Message + "\n"
	}

	// Obtener el prompt base
	prompt, err := getPromptBase(db, chatID)
	if err != nil {
		prompt = "Eres un asistente que comprime conversaciones de forma ultra-condensada. Extrae SOLO los puntos clave, decisiones y contexto esencial. Máximo 200 palabras."
	} else {
		prompt = prompt + "\n\nAhora resume esto de forma ULTRA-CONDENSADA (máximo 200 palabras), extrayendo SOLO lo esencial para mantener el contexto."
	}

	// Intentar con GEMINI
	compressedText, err := waifuSummaryGEMINI(textToCompress, prompt)
	if err != nil {
		log.Printf("Error con GEMINI al comprimir: %v", err)
	}

	// Si falla, intentar con GROQ
	if compressedText == "" {
		compressedText, err = groqIA(textToCompress, prompt)
		if err != nil {
			log.Printf("Error con GROQ al comprimir: %v", err)
			return nil // Salir si ambos fallan
		}
	}

	if compressedText == "" {
		return nil // No se pudo comprimir
	}

	// Guardar el nuevo contexto comprimido
	if err := insertCompressedContext(db, chatID, compressedText); err != nil {
		log.Printf("Error al guardar contexto comprimido: %v", err)
	}

	// Limpiar los mensajes sin procesar
	Clear(db, chatID)

	log.Printf("Contexto comprimido exitosamente. Nuevos tokens estimados: %d", estimateTokens(compressedText))

	return nil
}
