package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rturovtsev/telegram-bot-reminder/internal/handler"
	"github.com/rturovtsev/telegram-bot-reminder/internal/storage"
)

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	env := os.Getenv("ENV")

	if botToken == "" {
		log.Panic("BOT_TOKEN environment variable not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	var db *sql.DB

	if env == "dev" {
		bot.Debug = true
		db = storage.InitDB("reminders.db")
		log.Printf("Authorized on account %s", bot.Self.UserName)
	} else {
		bot.Debug = false
		db = storage.InitDB("/app/data/reminders.db")
	}

	defer db.Close()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go checkReminders(bot, db)

	for update := range updates {
		// Обрабатываем callback от inline-кнопок
		if update.CallbackQuery != nil {
			handler.HandleCallback(bot, update.CallbackQuery, db)
			continue
		}

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

		rows, err := db.Query("SELECT id, chat_id, text, repeat_type FROM reminders WHERE datetime < ?", now)
		if err != nil {
			log.Println(err)
			continue
		}
		defer rows.Close()

		var remindersToDelete []int
		var remindersToUpdate []struct {
			id           int
			nextDatetime time.Time
		}

		for rows.Next() {
			var id int
			var chatID int64
			var text string
			var repeatType string
			if err = rows.Scan(&id, &chatID, &text, &repeatType); err != nil {
				log.Println(err)
				continue
			}

			bot.Send(tgbotapi.NewMessage(chatID, "Reminder: "+text))

			if repeatType == "none" {
				remindersToDelete = append(remindersToDelete, id)
			} else {
				nextTime := calculateNextRepeat(now, repeatType)
				if !nextTime.IsZero() {
					remindersToUpdate = append(remindersToUpdate, struct {
						id           int
						nextDatetime time.Time
					}{id, nextTime})
				} else {
					remindersToDelete = append(remindersToDelete, id)
				}
			}
		}

		if len(remindersToDelete) > 0 {
			statement := fmt.Sprintf("DELETE FROM reminders WHERE id IN (%s)",
				strings.Trim(strings.Join(strings.Fields(fmt.Sprint(remindersToDelete)), ","), "[]"))
			_, err = db.Exec(statement)
			if err != nil {
				log.Println(err)
			}
		}

		for _, reminder := range remindersToUpdate {
			_, err = db.Exec("UPDATE reminders SET datetime = ? WHERE id = ?", reminder.nextDatetime, reminder.id)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func calculateNextRepeat(currentTime time.Time, repeatType string) time.Time {
	switch repeatType {
	case "daily":
		return currentTime.Add(24 * time.Hour)
	case "weekly":
		return currentTime.Add(7 * 24 * time.Hour)
	case "monthly":
		return currentTime.AddDate(0, 1, 0)
	case "yearly":
		return currentTime.AddDate(1, 0, 0)
	default:
		return time.Time{}
	}
}
