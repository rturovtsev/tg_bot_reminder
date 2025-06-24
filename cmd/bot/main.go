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
		if update.Message == nil {
			continue
		}

		handler.HandleMessage(bot, update, db)
	}
}

func calculateNextRepeatTime(currentTime time.Time, repeatType string) time.Time {
	switch repeatType {
	case "daily":
		return currentTime.AddDate(0, 0, 1)
	case "weekly":
		return currentTime.AddDate(0, 0, 7)
	case "monthly":
		return currentTime.AddDate(0, 1, 0)
	default:
		return currentTime
	}
}

func checkReminders(bot *tgbotapi.BotAPI, db *sql.DB) {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()

		rows, err := db.Query("SELECT id, chat_id, text, repeat_type, repeat_enabled FROM reminders WHERE datetime < ?", now)
		if err != nil {
			log.Println(err)
			continue
		}
		defer rows.Close()

		var remindersToDelete []int
		var remindersToUpdate []struct {
			id       int
			chatID   int64
			text     string
			nextTime time.Time
		}

		for rows.Next() {
			var id int
			var chatID int64
			var text string
			var repeatType string
			var repeatEnabled bool
			if err = rows.Scan(&id, &chatID, &text, &repeatType, &repeatEnabled); err != nil {
				log.Println(err)
				continue
			}

			// Send reminder
			bot.Send(tgbotapi.NewMessage(chatID, "Напоминание: "+text))

			if repeatEnabled && repeatType != "none" {
				// Calculate next reminder time
				nextTime := calculateNextRepeatTime(now, repeatType)
				remindersToUpdate = append(remindersToUpdate, struct {
					id       int
					chatID   int64
					text     string
					nextTime time.Time
				}{id, chatID, text, nextTime})
			} else {
				// Mark for deletion if not repeating
				remindersToDelete = append(remindersToDelete, id)
			}
		}

		// Update repeating reminders with next time
		for _, reminder := range remindersToUpdate {
			_, err = db.Exec("UPDATE reminders SET datetime = ? WHERE id = ?", reminder.nextTime, reminder.id)
			if err != nil {
				log.Println("Error updating repeating reminder:", err)
			}
		}

		// Delete non-repeating reminders
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
