package state

import (
	"sync"
)

// Manager управляет состояниями пользователей
type Manager struct {
	mu     sync.RWMutex
	states map[int64]*UserData // telegramID -> UserData
}

// NewManager создаёт новый менеджер состояний
func NewManager() *Manager {
	return &Manager{
		states: make(map[int64]*UserData),
	}
}

// GetState получает текущее состояние пользователя
func (sm *Manager) GetState(telegramID int64) UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if userData, exists := sm.states[telegramID]; exists {
		return userData.State
	}
	return StateNone
}

// SetState устанавливает состояние пользователя
func (sm *Manager) SetState(telegramID int64, state UserState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state == StateNone {
		// Если состояние None, удаляем запись
		delete(sm.states, telegramID)
		return
	}

	if _, exists := sm.states[telegramID]; !exists {
		sm.states[telegramID] = &UserData{
			State: state,
			Data:  make(map[string]interface{}),
		}
	} else {
		sm.states[telegramID].State = state
	}
}

// GetData получает временные данные пользователя
func (sm *Manager) GetData(telegramID int64, key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if userData, exists := sm.states[telegramID]; exists {
		value, ok := userData.Data[key]
		return value, ok
	}
	return nil, false
}

// SetData устанавливает временные данные пользователя
func (sm *Manager) SetData(telegramID int64, key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[telegramID]; !exists {
		// Создаём запись если её нет
		sm.states[telegramID] = &UserData{
			State: StateNone,
			Data:  make(map[string]interface{}),
		}
	}
	sm.states[telegramID].Data[key] = value
}

// ClearState очищает состояние и данные пользователя
func (sm *Manager) ClearState(telegramID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, telegramID)
}

// GetAllData получает все временные данные пользователя
func (sm *Manager) GetAllData(telegramID int64) map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if userData, exists := sm.states[telegramID]; exists {
		// Возвращаем копию, чтобы избежать race condition
		dataCopy := make(map[string]interface{})
		for k, v := range userData.Data {
			dataCopy[k] = v
		}
		return dataCopy
	}
	return nil
}
