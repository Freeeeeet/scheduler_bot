package common

import "errors"

// Общие ошибки для обработчиков
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrNotATeacher       = errors.New("user is not a teacher")
	ErrSubjectNotFound   = errors.New("subject not found")
	ErrNotSubjectOwner   = errors.New("user is not the owner of this subject")
	ErrNoMessage         = errors.New("no message in callback")
	ErrInvalidFormat     = errors.New("invalid callback format")
	ErrSlotNotFound      = errors.New("slot not found")
	ErrBookingNotFound   = errors.New("booking not found")
	ErrRecurringNotFound = errors.New("recurring schedule not found")
)

// ErrorMessage возвращает пользовательское сообщение для ошибки
func ErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrUserNotFound):
		return "❌ Пользователь не найден. Используйте /start"
	case errors.Is(err, ErrNotATeacher):
		return "❌ Эта функция доступна только учителям"
	case errors.Is(err, ErrSubjectNotFound):
		return "❌ Предмет не найден"
	case errors.Is(err, ErrNotSubjectOwner):
		return "❌ У вас нет доступа к этому предмету"
	case errors.Is(err, ErrNoMessage):
		return "❌ Ошибка обработки сообщения"
	case errors.Is(err, ErrInvalidFormat):
		return "❌ Неверный формат данных"
	case errors.Is(err, ErrSlotNotFound):
		return "❌ Слот не найден"
	case errors.Is(err, ErrBookingNotFound):
		return "❌ Бронирование не найдено"
	case errors.Is(err, ErrRecurringNotFound):
		return "❌ Расписание не найдено"
	default:
		return "❌ Произошла ошибка"
	}
}
