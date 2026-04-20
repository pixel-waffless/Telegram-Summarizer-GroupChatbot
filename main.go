package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"google.golang.org/genai"
	_ "modernc.org/sqlite"
)

func main() {
	// initialize database
	db, err := sql.Open("sqlite", "/app/data/db.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Create table database
	createTable(db)

	// Cargar variables de entorno desde el archivo .env
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró el archivo .env")
		log.Println("Se usará la variable de entorno")
	} else {
		log.Println("Archivo .env cargado")
	}

	// Variables de entorno
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	debugMode := os.Getenv("GO_ENV") == "development"

	if botToken == "" {
		log.Println("El TELEGRAM_BOT_TOKEN no se encontró")
	}

	log.Println("El TELEGRAM_BOT_TOKEN se cargó")

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = debugMode
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Main bot loop to respond
	for update := range updates {

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		userName := update.Message.From.UserName
		if userName == "" {
			userName = update.Message.From.FirstName
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ReplyToMessageID = update.Message.MessageID

		// Add messages and ignore /summary command
		if update.Message.Text != "/summary" && !update.Message.IsCommand() {
			userName := update.Message.From.FirstName
			message := update.Message.Text
			insertMessage(db, chatID, MessageMember{Name: userName, Message: message})
			// Automatically check and compress context if necessary
			go autoCompressContext(db, chatID)
		}

		command := strings.ToLower(update.Message.Command())

		if !update.Message.IsCommand() {
			continue
		}
		// Handle commands
		switch command {
		case "summary":
			textMessage := GetFormattedMessages(db, chatID, 300)
			prompt, err := getPromptBase(db, chatID)
			if err != nil {
				log.Printf("Error al obtener el prompt base: %v", err)
				prompt = "Eres un asistente que resume conversaciones de Telegram de forma breve y concisa. Resumes la conversación sin omitir detalles importantes, pero sin ser demasiado extenso. El resumen debe ser fácil de leer y entender."
			}
			// Add context from last summary
			lastSummary, err := getLastSummary(db, chatID)
			if err == nil && lastSummary != "" {
				prompt += "\nPrevious conversation context: " + lastSummary
			}
			// Add compressed context if exists
			compressedCtx, err := getCompressedContext(db, chatID)
			if err == nil && compressedCtx != "" {
				prompt += "\nCompressed context from previous conversations:\n" + compressedCtx
			}
			if textMessage == "" {
				msg.Text = "Eh no hay mensajes que resumir..."
				bot.Send(msg)
				continue
			}

			// Try with GEMINI
			summary, err := waifuSummaryGEMINI(textMessage, prompt)
			if err != nil {
				log.Printf("Error with GEMINI: %v", err)
			}

			// If fails or empty, try with GROQ
			if summary == "" {
				summary, err = groqIA(textMessage, prompt)
				if err != nil {
					log.Printf("Error with GROQ: %v", err)
				}

				if summary == "" {
					summary, err = waifuSummaryGIPITI(textMessage, prompt)
					if err != nil {
						msg.Text = "Eh, no quiero resumir nada largarte. **Se duerme**."
						bot.Send(msg)
						continue
					}
					if summary == "" {
						msg.Text = "No hay nada para ver aquí... Fuun, tsumannai..."
						bot.Send(msg)
						continue

					}

				}
			}

			msg.Text = summary
			msg.ParseMode = ""
			bot.Send(msg)
			// Save summary for future context
			insertSummary(db, chatID, summary)
			// Clear messages after summary
			Clear(db, chatID)
		case "help":
			media := []interface{}{
				tgbotapi.NewInputMediaPhoto(
					tgbotapi.FileURL("https://i.pinimg.com/736x/5b/49/91/5b499161daba947d434f1b8cd41530fd.jpg"),
				),
			}
			photo := media[0].(tgbotapi.InputMediaPhoto)
			photo.Caption = helpText
			photo.ParseMode = "Markdown"
			media[0] = photo
			mediaGroup := tgbotapi.NewMediaGroup(update.Message.Chat.ID, media)
			mediaGroup.ReplyToMessageID = update.Message.MessageID

			_, err := bot.SendMediaGroup(mediaGroup)
			if err != nil {
				log.Printf("Error al enviar el mensaje con foto: %v", err)

				// Opcional: si falla el media group, envía solo el texto como respaldo
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, helpText)
				msg.ParseMode = "Markdown"
				bot.Send(msg)
			}

		case "getstats":
			stats, _ := GetStats(db, chatID)
			msg.Text = stats
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		case "clear":
			Clear(db, chatID)
			msg.Text = `Ya me auto formateé la cabeza, ahora a mimir... **Se duerme**`
			bot.Send(msg)
		case "promptbase":
			if update.Message.From.ID != 7046723187 {
				msg.Text = "No tienes permiso para usar este comando"
				bot.Send(msg)
				continue
			}
			insertPromptBase(db, chatID, update.Message.Text)
			msg.Text = "Prompt guardado con éxito"
			msg.ParseMode = ""
			bot.Send(msg)
		case "ask":
			// Remove command from text
			commandText := "/" + update.Message.Command()
			inputText := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, commandText))
			input := update.Message.From.FirstName + ": " + inputText
			prompt, err := getPromptBase(db, chatID)
			textContext := GetFormattedMessages(db, chatID, 50)
			if err != nil {
				log.Printf("Error al obtener el prompt base: %v", err)
				prompt = "Eres un asistente que responde preguntas de forma breve y concisa. Respondes sin omitir detalles importantes, pero sin ser demasiado extenso. El resumen debe ser fácil de leer y entender."
			}
			// Agregar contexto comprimido si existe
			compressedCtx, err := getCompressedContext(db, chatID)
			if err == nil && compressedCtx != "" {
				textContext = "Contexto comprimido:\n" + compressedCtx + "\n\nMensajes recientes:\n" + textContext
			}
			prompt += "\nEn esto se basa tu personalidad:" + prompt + "este es el contexto del chat no tienes que responder " + textContext + "a esto en base a lo anterior responde a pregunta de manera resumida: "

			// Intento con GEMINI
			answer, err := waifuSummaryGEMINI(input, prompt)
			if err != nil {
				log.Printf("Error con GEMINI: %v", err)
			}

			// Intento con GROQ
			if answer == "" {
				answer, err = groqIA(input, prompt)
				if err != nil {
					log.Printf("Error con GROQ: %v", err)
					waifuSummaryGIPITI(input, prompt)
				}

				if answer == "" {
					msg.Text = "No hay nada para ver aquí... Fuun, tsumannai..."
					bot.Send(msg)
					continue
				}
			}

			username := update.Message.From.UserName
			if username != "" {
				msg.Text = "@" + username + " " + answer
			} else {
				msg.Text = answer
			}

			msg.ParseMode = ""
			bot.Send(msg)
			// Verificar y comprimir contexto automáticamente si es necesario
			go autoCompressContext(db, chatID)
		default:
			log.Println("No hay comando válido")
		}

	}
}

// Función para llamar a la API de gemini
func waifuSummaryGEMINI(message string, prompt string) (string, error) {
	// Verificar que la variable de entorno exista
	GEMINI_API_KEY := os.Getenv("GEMINI_API_KEY")
	if GEMINI_API_KEY == "" {
		log.Println("El GEMINI_API_KEY no se encontró")
		return "", nil
	}
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return "Error al crear el cliente GEMINI", err
	}

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash",
		genai.Text(prompt+"\n\n"+message),
		nil,
	)

	if err != nil {
		log.Printf("Error al generar el resumen con GEMINI: %v", err)
		return "", err
	}

	return result.Text(), nil
}

// Función para llamar a la API de VENICE
func waifuSummaryGIPITI(message string, prompt string) (string, error) {
	// Verificar que la variable de entorno exista
	VENICE_API_KEY := os.Getenv("VENICE_API_KEY")
	if VENICE_API_KEY == "" {
		log.Println("El VENICE_API_KEY no se encontró")
		return "", nil
	}

	client := openai.NewClient(
		option.WithAPIKey(VENICE_API_KEY),
		option.WithBaseURL("https://api.venice.ai/api/v1"))

	ctx := context.Background()

	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: *openai.StringPtr("e2ee-gpt-oss-20b-p:include_venice_system_prompt=false"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt),
			openai.UserMessage(message),
		},
		Temperature: openai.Float(0.7),
	},
	)

	if err != nil {
		log.Printf("Error al generar el resumen con GIPITI: %v", err)
		return "", err
	}

	return chatCompletion.Choices[0].Message.Content, nil
}

func groqIA(message string, prompt string) (string, error) {
	BASE_URL := "https://api.groq.com/openai/v1/chat/completions"
	GROQ_API_KEY := os.Getenv("GROQ_API_KEY")

	if GROQ_API_KEY == "" {
		log.Println("No se encontro la GROQ_API_KEY")
	}

	jsonPayload := map[string]interface{}{
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": prompt},
			{
				"role":    "user",
				"content": message,
			},
		},
		"model":                 "moonshotai/kimi-k2-instruct-0905",
		"temperature":           1,
		"max_completion_tokens": 8192,
		"top_p":                 1,
		"stream":                false,
		"stop":                  nil,
	}

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		fmt.Println("Error serializing JSON:", err)
		return "", err
	}

	bodyData := bytes.NewBuffer(jsonData)

	// Crear la solicitud HTTP
	req, err := http.NewRequest("POST", BASE_URL, bodyData)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+GROQ_API_KEY)

	// Tiempo de espera para la solicitud
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Obtenemos la respuesta
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error al obtener la respuesta:", err)
		return "", err
	}
	defer resp.Body.Close()

	// Verificar el código de estado
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: estado %d, body: %s\n", resp.StatusCode, string(body))
		return "", err

	}

	// Leer el cuerpo de la respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return "", err

	}

	// Parse response
	var post struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &post); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return "", err

	}

	return post.Choices[0].Message.Content, nil
}
