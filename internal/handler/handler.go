package handler

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/ru"
	"log"
	"strings"
	"time"
)

func HandleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	if strings.HasPrefix(text, "/remindme") {
		w := when.New(nil)
		w.Add(ru.All...)
		w.Add(common.All...)
		w.SetOptions(&rules.Options{
			Distance:     10,
			MatchByOrder: true,
		})

		r, err := w.Parse(text, time.Now())

		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Invalid arguments. Usage: /remindme <time> <message>"))
			return
		}

		dateTime, reminderSource := r.Time, r.Source

		_, err = db.Exec("INSERT INTO reminders (chat_id, text, datetime) VALUES (?, ?, ?)", chatID, reminderSource, dateTime)
		if err != nil {
			log.Println(err)
			bot.Send(tgbotapi.NewMessage(chatID, "Failed to add reminder"))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Reminder set: %s at %s", reminderSource, dateTime)))
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
