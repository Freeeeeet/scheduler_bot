package keyboard

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

// BackButton создаёт кнопку "Назад"
func BackButton(callbackData string) models.InlineKeyboardButton {
	return Button("⬅️ Назад", callbackData)
}

// BackToMainButton создаёт кнопку "В главное меню"
func BackToMainButton() models.InlineKeyboardButton {
	return Button("🏠 В главное меню", "back_to_main")
}

// BackToSubjectsButton создаёт кнопку "К списку предметов"
func BackToSubjectsButton() models.InlineKeyboardButton {
	return Button("⬅️ К списку предметов", "back_to_subjects")
}

// BackToMyScheduleButton создаёт кнопку "К моему расписанию"
func BackToMyScheduleButton() models.InlineKeyboardButton {
	return Button("⬅️ К расписанию", "back_to_myschedule")
}

// CancelButton создаёт кнопку "Отмена"
func CancelButton(callbackData string) models.InlineKeyboardButton {
	return Button("❌ Отмена", callbackData)
}

// ConfirmButton создаёт кнопку "Подтвердить"
func ConfirmButton(callbackData string) models.InlineKeyboardButton {
	return Button("✅ Подтвердить", callbackData)
}

// YesNoButtons создаёт два ряда с кнопками Да/Нет
func YesNoButtons(yesCallback, noCallback string) [][]models.InlineKeyboardButton {
	return [][]models.InlineKeyboardButton{
		{
			Button("✅ Да", yesCallback),
			Button("❌ Нет", noCallback),
		},
	}
}

// ConfirmCancelButtons создаёт два ряда с кнопками Подтвердить/Отмена
func ConfirmCancelButtons(confirmCallback, cancelCallback string) [][]models.InlineKeyboardButton {
	return [][]models.InlineKeyboardButton{
		{
			ConfirmButton(confirmCallback),
			CancelButton(cancelCallback),
		},
	}
}

// BackRow создаёт ряд с кнопкой "Назад"
func BackRow(callbackData string) []models.InlineKeyboardButton {
	return []models.InlineKeyboardButton{BackButton(callbackData)}
}

// AddBackButton добавляет кнопку "Назад" к builder
func (b *Builder) AddBackButton(callbackData string) *Builder {
	return b.Row(BackButton(callbackData))
}

// AddBackToMainButton добавляет кнопку "В главное меню" к builder
func (b *Builder) AddBackToMainButton() *Builder {
	return b.Row(BackToMainButton())
}

// AddBackToSubjectsButton добавляет кнопку "К списку предметов" к builder
func (b *Builder) AddBackToSubjectsButton() *Builder {
	return b.Row(BackToSubjectsButton())
}

// ViewScheduleButton создаёт кнопку "Управление расписанием"
func ViewScheduleButton() models.InlineKeyboardButton {
	return Button("📅 Управление расписанием", "view_schedule")
}

// CreateSlotButton создаёт кнопку "Создать слот"
func CreateSlotButton(subjectID int64) models.InlineKeyboardButton {
	return Button("➕ Создать слот", fmt.Sprintf("create_slots:%d", subjectID))
}

// EditButton создаёт кнопку "Редактировать"
func EditButton(callbackData string) models.InlineKeyboardButton {
	return Button("✏️ Редактировать", callbackData)
}

// DeleteButton создаёт кнопку "Удалить"
func DeleteButton(callbackData string) models.InlineKeyboardButton {
	return Button("🗑 Удалить", callbackData)
}
