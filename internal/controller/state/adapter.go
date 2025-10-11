package state

import (
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
)

// Adapter адаптирует state.Manager к интерфейсу callbacktypes.StateManager
type Adapter struct {
	sm *Manager
}

// NewAdapter создает адаптер для Manager
func NewAdapter(sm *Manager) *Adapter {
	return &Adapter{sm: sm}
}

// GetState получает текущее состояние пользователя
func (a *Adapter) GetState(telegramID int64) callbacktypes.UserState {
	// Преобразуем state.UserState в callbacktypes.UserState
	return callbacktypes.UserState(a.sm.GetState(telegramID))
}

// SetState устанавливает состояние пользователя
func (a *Adapter) SetState(telegramID int64, state callbacktypes.UserState) {
	// Преобразуем callbacktypes.UserState в state.UserState
	a.sm.SetState(telegramID, UserState(state))
}

// GetData получает временные данные пользователя
func (a *Adapter) GetData(telegramID int64, key string) (interface{}, bool) {
	return a.sm.GetData(telegramID, key)
}

// SetData устанавливает временные данные пользователя
func (a *Adapter) SetData(telegramID int64, key string, value interface{}) {
	a.sm.SetData(telegramID, key, value)
}

// ClearState очищает состояние и данные пользователя
func (a *Adapter) ClearState(telegramID int64) {
	a.sm.ClearState(telegramID)
}

// GetAllData получает все временные данные пользователя
func (a *Adapter) GetAllData(telegramID int64) map[string]interface{} {
	return a.sm.GetAllData(telegramID)
}
