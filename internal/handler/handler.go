package handler

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
	"time"
)

func HandleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	if strings.HasPrefix(text, "/remindme") {
		args := strings.SplitN(text, " ", 4)
		if len(args) < 4 {
			bot.Send(tgbotapi.NewMessage(chatID, "Invalid arguments. Usage: /remindme <time> <message>"))
			return
		}

		dateTimeStr, reminderTime, reminderText := args[1], args[2], args[3]
		dateTime, err := time.Parse("2006-01-02 15:04", dateTimeStr+" "+reminderTime)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Invalid time format. Use YYYY-MM-DD HH:MM"))
			return
		}

		_, err = db.Exec("INSERT INTO reminders (chat_id, text, datetime) VALUES (?, ?, ?)", chatID, reminderText, dateTime)
		if err != nil {
			log.Println(err)
			bot.Send(tgbotapi.NewMessage(chatID, "Failed to add reminder"))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Reminder set: %s at %s", reminderText, dateTimeStr)))
		}
	} else if strings.HasPrefix(text, "/listreminders") {
		rows, err := db.Query("SELECT text, datetime FROM reminders WHERE chat_id = ?", chatID)
		if err != nil {
			log.Println(err)
			bot.Send(tgbotapi.NewMessage(chatID, "Failed to list reminders"))
			return
		}
		defer rows.Close()

		var response string
		for rows.Next() {
			var reminderText string
			var dateTime time.Time
			if err = rows.Scan(&reminderText, &dateTime); err != nil {
				log.Println(err)
				continue
			}
			response += fmt.Sprintf("- %s at %s\n", reminderText, dateTime.Format("2006-01-02 15:04"))
		}

		if response == "" {
			response = "No active reminders"
		}

		bot.Send(tgbotapi.NewMessage(chatID, response))
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, "Unknown command"))
	}

}
