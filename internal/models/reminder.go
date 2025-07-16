package models

import "time"

type Reminder struct {
	ID         int
	ChatID     int64
	Text       string
	Datetime   time.Time
	RepeatType string
}
