package keyboard

import "github.com/go-telegram/bot/models"

// Builder упрощает создание inline клавиатур
type Builder struct {
	rows [][]models.InlineKeyboardButton
}

// NewBuilder создаёт новый builder клавиатуры
func NewBuilder() *Builder {
	return &Builder{
		rows: make([][]models.InlineKeyboardButton, 0),
	}
}

// Row добавляет новый ряд кнопок
func (b *Builder) Row(buttons ...models.InlineKeyboardButton) *Builder {
	if len(buttons) > 0 {
		b.rows = append(b.rows, buttons)
	}
	return b
}

// Button создаёт кнопку
func Button(text, callbackData string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

// URLButton создаёт кнопку с URL
func URLButton(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

// AddRow добавляет полностью готовый ряд кнопок
func (b *Builder) AddRow(row []models.InlineKeyboardButton) *Builder {
	if len(row) > 0 {
		b.rows = append(b.rows, row)
	}
	return b
}

// AddRows добавляет несколько рядов кнопок
func (b *Builder) AddRows(rows [][]models.InlineKeyboardButton) *Builder {
	b.rows = append(b.rows, rows...)
	return b
}

// Build создаёт финальную клавиатуру
func (b *Builder) Build() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: b.rows,
	}
}

// Empty возвращает пустую клавиатуру (без кнопок)
func Empty() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{},
	}
}
