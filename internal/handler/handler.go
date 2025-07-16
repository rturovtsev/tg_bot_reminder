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

	if strings.HasPrefix(text, "/start") {
		_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Бот запущен. Доступные команды: /list\n\nДля создания напоминаний:\n• Обычные: 'сегодня в 14:00 встреча'\n• Повторяющиеся: 'ежедневно в 9:00 зарядка', 'еженедельно в понедельник совещание', 'ежемесячно 1 числа оплата'"))
	} else if strings.HasPrefix(text, "/list") {
		rows, err := db.Query("SELECT text, datetime, repeat_type FROM reminders WHERE chat_id = ?", chatID)
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
			var reminderText string
			var dateTime time.Time
			var repeatType string
			if err = rows.Scan(&reminderText, &dateTime, &repeatType); err != nil {
				log.Println(err)
				continue
			}
			repeatLabel := getRepeatLabel(repeatType)
			response += fmt.Sprintf("- <blockquote>%s</blockquote> Время срабатывания: %s %s\n", reminderText, dateTime.Format("2006-01-02 15:04"), repeatLabel)
		}

		if response == "" {
			response = "Нет активных напоминаний"
		}

		txt := tgbotapi.NewMessage(chatID, response)
		txt.ParseMode = tgbotapi.ModeHTML

		_, _ = bot.Send(txt)
	} else {
		repeatType := parseRepeatType(text)
		cleanedText := removeRepeatKeywords(text)

		w := when.New(nil)
		w.Add(ru.All...)
		w.Add(common.All...)
		w.SetOptions(&rules.Options{
			Distance:     10,
			MatchByOrder: true,
		})

		r, err := w.Parse(cleanedText, time.Now())

		if err != nil || r == nil {
			formatExample := "Ошибка формата времени. Возможные варианты:\n - сегодня в 11:10 {ваш_текст}\n - в пятницу после обеда {ваш_текст}\n - 14:00 следующего вторника {ваш_текст}\n - в следующую среду в 12:25 {ваш_текст}\n - ежедневно в 9:00 {ваш_текст}\n - еженедельно в понедельник {ваш_текст}\n - ежемесячно 1 числа {ваш_текст}"

			_, _ = bot.Send(tgbotapi.NewMessage(chatID, formatExample))
			return
		}

		dateTime, reminderSource := r.Time, r.Source
		reminderSource = strings.Replace(reminderSource, "/add", "", -1)

		_, err = db.Exec("INSERT INTO reminders (chat_id, text, datetime, repeat_type) VALUES (?, ?, ?, ?)", chatID, reminderSource, dateTime, repeatType)
		if err != nil {
			log.Println(err)
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения напоминания"))
		} else {
			repeatLabel := getRepeatLabel(repeatType)
			txt := tgbotapi.NewMessage(chatID, fmt.Sprintf("Установленное напоминание: <blockquote>%s</blockquote> Время срабатывания: %s %s", reminderSource, dateTime.Format("2006-01-02 15:04"), repeatLabel))
			txt.ParseMode = tgbotapi.ModeHTML
			_, _ = bot.Send(txt)
		}
	}
}

func parseRepeatType(text string) string {
	lowerText := strings.ToLower(text)
	
	if strings.Contains(lowerText, "ежедневно") || strings.Contains(lowerText, "каждый день") {
		return "daily"
	}
	if strings.Contains(lowerText, "еженедельно") || strings.Contains(lowerText, "каждую неделю") {
		return "weekly"
	}
	if strings.Contains(lowerText, "ежемесячно") || strings.Contains(lowerText, "каждый месяц") {
		return "monthly"
	}
	if strings.Contains(lowerText, "ежегодно") || strings.Contains(lowerText, "каждый год") {
		return "yearly"
	}
	
	return "none"
}

func removeRepeatKeywords(text string) string {
	keywords := []string{
		"ежедневно", "каждый день",
		"еженедельно", "каждую неделю",
		"ежемесячно", "каждый месяц",
		"ежегодно", "каждый год",
	}
	
	cleanedText := text
	for _, keyword := range keywords {
		cleanedText = strings.ReplaceAll(strings.ToLower(cleanedText), keyword, "")
	}
	
	return strings.TrimSpace(cleanedText)
}

func getRepeatLabel(repeatType string) string {
	switch repeatType {
	case "daily":
		return "(повторяется ежедневно)"
	case "weekly":
		return "(повторяется еженедельно)"
	case "monthly":
		return "(повторяется ежемесячно)"
	case "yearly":
		return "(повторяется ежегодно)"
	default:
		return ""
	}
}
