package main

import (
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendStickerHandle(update tgbotapi.Update, emoji string, bot *tgbotapi.BotAPI, chatID int64) {

	sets, err := bot.GetStickerSet(
		tgbotapi.GetStickerSetConfig{Name: "manystickskomi_by_ilq6_gbot"},
	)

	if err != nil {
		log.Println("Error getting sticker set:", err)
		return
	}

	if len(sets.Stickers) == 0 {
		log.Println("Sticker set is empty")
		return
	}

	for _, sticker := range sets.Stickers {
		if sticker.Emoji == emoji {
			sendSticker(chatID, update.Message.MessageID, sticker.FileID, bot)
			return
		}

	}

	randomIndex := time.Now().UnixNano() % int64(len(sets.Stickers))
	randomSticker := sets.Stickers[randomIndex]
	sendSticker(chatID, update.Message.MessageID, randomSticker.FileID, bot)
}

func sendSticker(chatID int64, messageID int, stickerID string, bot *tgbotapi.BotAPI) {
	sticker := tgbotapi.NewSticker(chatID, tgbotapi.FileID(stickerID))
	sticker.ReplyToMessageID = messageID
	bot.Send(sticker)
	return
}

func extractEmoji(text string) string {
	matches := ""
	for _, emoji := range strings.Split(emojiList, "") {
		if strings.Contains(text, emoji) {
			matches = emoji
			return matches
		}
	}
	return ""
}
