package main

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rturovtsev/telegram-bot-reminder/internal/handler"
	"github.com/rturovtsev/telegram-bot-reminder/internal/storage"
	"log"
	"os"
	"strings"
	"time"
)

var (
	botToken = os.Getenv("BOT_TOKEN")
)

func main() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	db := storage.InitDB("reminders.db")
	defer db.Close()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go checkReminders(bot, db)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		handler.HandleMessage(bot, update, db)
	}
}

func checkReminders(bot *tgbotapi.BotAPI, db *sql.DB) {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()

		rows, err := db.Query("SELECT id, chat_id, text FROM reminders WHERE datetime < ?", now)
		if err != nil {
			log.Println(err)
			continue
		}
		defer rows.Close()

		var remindersToDelete []int

		for rows.Next() {
			var id int
			var chatID int64
			var text string
			if err = rows.Scan(&id, &chatID, &text); err != nil {
				log.Println(err)
				continue
			}

			bot.Send(tgbotapi.NewMessage(chatID, "Reminder: "+text))
			remindersToDelete = append(remindersToDelete, id)
		}

		if len(remindersToDelete) > 0 {
			statement := fmt.Sprintf("DELETE FROM reminders WHERE id IN (%s)",
				strings.Trim(strings.Join(strings.Fields(fmt.Sprint(remindersToDelete)), ","), "[]"))
			_, err = db.Exec(statement)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
