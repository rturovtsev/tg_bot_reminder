package handler

import "sync"

// UserState хранит состояние пользователя при редактировании напоминания
type UserState struct {
	Action     string // "edit" или другие действия
	ReminderID int    // ID напоминания для редактирования
}

var (
	userStates = make(map[int64]*UserState)
	stateMutex sync.RWMutex
)

// SetUserState устанавливает состояние для пользователя
func SetUserState(chatID int64, state *UserState) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	userStates[chatID] = state
}

// GetUserState получает состояние пользователя
func GetUserState(chatID int64) (*UserState, bool) {
	stateMutex.RLock()
	defer stateMutex.RUnlock()
	state, exists := userStates[chatID]
	return state, exists
}

// ClearUserState очищает состояние пользователя
func ClearUserState(chatID int64) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	delete(userStates, chatID)
}
