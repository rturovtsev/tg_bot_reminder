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
		_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Бот запущен. Доступные команды: /add, /list"))
	} else if strings.HasPrefix(text, "/list") {
		rows, err := db.Query("SELECT text, datetime FROM reminders WHERE chat_id = ?", chatID)
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
			if err = rows.Scan(&reminderText, &dateTime); err != nil {
				log.Println(err)
				continue
			}
			response += fmt.Sprintf("- <blockquote>%s</blockquote> Время срабатывания: %s\n", reminderText, dateTime.Format("2006-01-02 15:04"))
		}

		if response == "" {
			response = "Нет активных напоминаний"
		}

		txt := tgbotapi.NewMessage(chatID, response)
		txt.ParseMode = tgbotapi.ModeHTML

		_, _ = bot.Send(txt)
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

		_, err = db.Exec("INSERT INTO reminders (chat_id, text, datetime) VALUES (?, ?, ?)", chatID, reminderSource, dateTime)
		if err != nil {
			log.Println(err)
			_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения напоминания"))
		} else {
			txt := tgbotapi.NewMessage(chatID, fmt.Sprintf("Установленное напоминание: <blockquote>%s</blockquote> Время срабатывания: %s", reminderSource, dateTime.Format("2006-01-02 15:04")))
			txt.ParseMode = tgbotapi.ModeHTML
			_, _ = bot.Send(txt)
		}
	}
}
