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

func detectRepeatType(text string) (string, bool) {
	text = strings.ToLower(text)

	dailyKeywords := []string{"каждый день", "ежедневно", "каждые сутки", "daily"}
	weeklyKeywords := []string{"каждую неделю", "еженедельно", "weekly"}
	monthlyKeywords := []string{"каждый месяц", "ежемесячно", "monthly"}

	for _, keyword := range dailyKeywords {
		if strings.Contains(text, keyword) {
			return "daily", true
		}
	}

	for _, keyword := range weeklyKeywords {
		if strings.Contains(text, keyword) {
			return "weekly", true
		}
	}

	for _, keyword := range monthlyKeywords {
		if strings.Contains(text, keyword) {
			return "monthly", true
		}
	}

	return "none", false
}

func HandleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	if strings.HasPrefix(text, "/start") {
		_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Бот запущен. Доступные команды: /list, /delete\n\nПримеры:\n- сегодня в 15:00 встреча\n- каждый день в 9:00 принять витамины\n- ежемесячно 1 числа в 10:00 оплатить квартиру"))
	} else if strings.HasPrefix(text, "/list") {
		rows, err := db.Query("SELECT id, text, datetime, repeat_type, repeat_enabled FROM reminders WHERE chat_id = ?", chatID)
		if err != nil {
			log.Println(err)
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Ошибка получения списка напоминаний"))
			return
		}
		defer func(rows *sql.Rows) {
			err = rows.Close()
			if err != nil {
				log.Println(err)
			}
		}(rows)

		var response string
		for rows.Next() {
			var id int
			var reminderText string
			var dateTime time.Time
			var repeatType string
			var repeatEnabled bool
			if err = rows.Scan(&id, &reminderText, &dateTime, &repeatType, &repeatEnabled); err != nil {
				log.Println(err)
				continue
			}

			repeatStr := ""
			if repeatEnabled {
				switch repeatType {
				case "daily":
					repeatStr = " (ежедневно)"
				case "weekly":
					repeatStr = " (еженедельно)"
				case "monthly":
					repeatStr = " (ежемесячно)"
				}
			}

			response += fmt.Sprintf("ID: %d - <blockquote>%s</blockquote>%s Время срабатывания: %s\n",
				id, reminderText, repeatStr, dateTime.Format("2006-01-02 15:04"))
		}

		if response == "" {
			response = "Нет активных напоминаний"
		}

		txt := tgbotapi.NewMessage(chatID, response)
		txt.ParseMode = tgbotapi.ModeHTML

		_, _ = bot.Send(txt)
	} else if strings.HasPrefix(text, "/delete ") {
		idStr := strings.TrimPrefix(text, "/delete ")
		idStr = strings.TrimSpace(idStr)

		if idStr == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Использование: /delete [ID]\nПосмотрите ID напоминаний через /list"))
			return
		}

		result, err := db.Exec("DELETE FROM reminders WHERE id = ? AND chat_id = ?", idStr, chatID)
		if err != nil {
			log.Println(err)
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Ошибка удаления напоминания"))
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Напоминание с таким ID не найдено"))
		} else {
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Напоминание удалено"))
		}
	} else {
		w := when.New(nil)
		w.Add(ru.All...)
		w.Add(common.All...)
		w.SetOptions(&rules.Options{
			Distance:     10,
			MatchByOrder: true,
		})

		r, err := w.Parse(text, time.Now())

		if err != nil || r == nil {
			formatExample := "Ошибка формата времени. Возможные варианты:\n - сегодня в 11:10 {ваш_текст}\n - в пятницу после обеда {ваш_текст}\n - 14:00 следующего вторника {ваш_текст}\n - в следующую среду в 12:25 {ваш_текст}"

			_, _ = bot.Send(tgbotapi.NewMessage(chatID, formatExample))
			return
		}

		dateTime, reminderSource := r.Time, r.Source
		reminderSource = strings.Replace(reminderSource, "/add", "", -1)

		// Detect repeat type
		repeatType, isRepeating := detectRepeatType(text)

		// Clean reminder text from repeat keywords
		cleanText := reminderSource
		if isRepeating {
			repeatKeywords := []string{"каждый день", "ежедневно", "каждые сутки", "daily",
				"каждую неделю", "еженедельно", "weekly",
				"каждый месяц", "ежемесячно", "monthly"}
			for _, keyword := range repeatKeywords {
				cleanText = strings.ReplaceAll(strings.ToLower(cleanText), keyword, "")
			}
			cleanText = strings.TrimSpace(cleanText)
		}

		_, err = db.Exec("INSERT INTO reminders (chat_id, text, datetime, repeat_type, repeat_enabled) VALUES (?, ?, ?, ?, ?)",
			chatID, cleanText, dateTime, repeatType, isRepeating)
		if err != nil {
			log.Println(err)
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения напоминания"))
		} else {
			var responseMsg string
			if isRepeating {
				var repeatText string
				switch repeatType {
				case "daily":
					repeatText = "ежедневно"
				case "weekly":
					repeatText = "еженедельно"
				case "monthly":
					repeatText = "ежемесячно"
				}
				responseMsg = fmt.Sprintf("Установленное повторяющееся напоминание (%s): <blockquote>%s</blockquote> Время срабатывания: %s",
					repeatText, cleanText, dateTime.Format("2006-01-02 15:04"))
			} else {
				responseMsg = fmt.Sprintf("Установленное напоминание: <blockquote>%s</blockquote> Время срабатывания: %s",
					cleanText, dateTime.Format("2006-01-02 15:04"))
			}

			txt := tgbotapi.NewMessage(chatID, responseMsg)
			txt.ParseMode = tgbotapi.ModeHTML
			_, _ = bot.Send(txt)
		}
	}
}
