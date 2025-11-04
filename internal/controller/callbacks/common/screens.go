package common

import (
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/go-telegram/bot/models"
)

// BuildEditSubjectScreen формирует экран редактирования предмета
func BuildEditSubjectScreen(subject *model.Subject) (string, *models.InlineKeyboardMarkup) {
	price := float64(subject.Price) / 100
	statusText := "Активен ✅"
	if !subject.IsActive {
		statusText = "Неактивен ⏸"
	}
	approvalText := "Нет ❌"
	if subject.RequiresBookingApproval {
		approvalText = "Да ✅"
	}

	text := fmt.Sprintf(
		"🛠 <b>Редактирование предмета</b>\n\n"+
			"📚 Название: %s\n"+
			"📝 Описание: %s\n"+
			"💰 Цена: %.2f ₽\n"+
			"⏱ Длительность: %d мин\n"+
			"⏳ Требуется одобрение: %s\n"+
			"📊 Статус: %s\n\n"+
			"Выберите, что хотите изменить:",
		subject.Name,
		subject.Description,
		price,
		subject.Duration,
		approvalText,
		statusText,
	)

	// Формируем текст для кнопок с текущим состоянием
	approvalButtonText := "⏳ Требуется одобрение: нет"
	if subject.RequiresBookingApproval {
		approvalButtonText = "⏳ Требуется одобрение: да"
	}

	statusButtonText := "📊 Статус: активен"
	if !subject.IsActive {
		statusButtonText = "📊 Статус: неактивен"
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📝 Название", CallbackData: fmt.Sprintf("edit_field_name:%d", subject.ID)},
				{Text: "📄 Описание", CallbackData: fmt.Sprintf("edit_field_desc:%d", subject.ID)},
			},
			{
				{Text: "💰 Цена", CallbackData: fmt.Sprintf("edit_field_price:%d", subject.ID)},
				{Text: "⏱ Длительность", CallbackData: fmt.Sprintf("edit_field_duration:%d", subject.ID)},
			},
			{
				{Text: approvalButtonText, CallbackData: fmt.Sprintf("toggle_approval:%d", subject.ID)},
			},
			{
				{Text: statusButtonText, CallbackData: fmt.Sprintf("toggle_subject:%d:edit", subject.ID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_subject:%d", subject.ID)},
			},
		},
	}

	return text, keyboard
}

// BuildViewSubjectScreen формирует экран просмотра предмета
func BuildViewSubjectScreen(subject *model.Subject) (string, *models.InlineKeyboardMarkup) {
	price := float64(subject.Price) / 100
	statusText := "✅ Активен"
	if !subject.IsActive {
		statusText = "⏸ Неактивен"
	}

	approvalText := "❌ Нет"
	if subject.RequiresBookingApproval {
		approvalText = "✅ Да"
	}

	text := fmt.Sprintf(
		"📚 <b>%s</b>\n\n"+
			"📝 Описание: %s\n"+
			"💰 Цена: %.2f ₽\n"+
			"⏱ Длительность: %d мин\n"+
			"📊 Статус: %s\n"+
			"⏳ Требуется одобрение: %s\n\n"+
			"Выберите действие:",
		subject.Name,
		subject.Description,
		price,
		subject.Duration,
		statusText,
		approvalText,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📅 Посмотреть расписание", CallbackData: fmt.Sprintf("view_schedule_calendar:%d", subject.ID)},
			},
			{
				{Text: "📊 Управление расписанием", CallbackData: fmt.Sprintf("subject_schedule:%d", subject.ID)},
			},
			{
				{Text: "✏️ Редактировать", CallbackData: fmt.Sprintf("edit_subject:%d", subject.ID)},
			},
			{
				{Text: "🗑 Удалить предмет", CallbackData: fmt.Sprintf("delete_subject:%d", subject.ID)},
			},
			{
				{Text: "⬅️ Назад к списку", CallbackData: "back_to_subjects"},
			},
		},
	}

	return text, keyboard
}

// BuildSubjectsListScreen формирует экран списка предметов с пагинацией
func BuildSubjectsListScreen(subjects []*model.Subject, page int) (string, *models.InlineKeyboardMarkup) {
	const pageSize = 10

	text := fmt.Sprintf("📚 Ваши предметы (всего: %d):\n\n", len(subjects))
	var buttons [][]models.InlineKeyboardButton

	// Вычисляем индексы для текущей страницы
	startIdx := page * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(subjects) {
		endIdx = len(subjects)
	}

	// Показываем предметы текущей страницы
	for i := startIdx; i < endIdx; i++ {
		subject := subjects[i]
		statusEmoji := "✅"
		statusText := "Активен"

		if !subject.IsActive {
			statusEmoji = "⏸"
			statusText = "Неактивен"
		}

		text += fmt.Sprintf(
			"%d. %s %s\n"+
				"   💰 Цена: %.2f ₽\n"+
				"   ⏱ Длительность: %d мин\n"+
				"   📝 %s\n"+
				"   Статус: %s\n\n",
			i+1,
			statusEmoji,
			subject.Name,
			float64(subject.Price)/100,
			subject.Duration,
			subject.Description,
			statusText,
		)

		// Кнопки для каждого предмета
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("📝 %s", subject.Name), CallbackData: fmt.Sprintf("view_subject:%d", subject.ID)},
			{Text: statusEmoji, CallbackData: fmt.Sprintf("toggle_subject:%d:list", subject.ID)},
		})
	}

	// Добавляем подсказку
	text += "\n💡 Совет: Создайте временные слоты через /myschedule чтобы студенты могли записываться!\n\n"

	// Кнопки пагинации
	totalPages := (len(subjects) + pageSize - 1) / pageSize
	if totalPages > 1 {
		var paginationButtons []models.InlineKeyboardButton

		// Кнопка "Предыдущая" только если не первая страница
		if page > 0 {
			paginationButtons = append(paginationButtons,
				models.InlineKeyboardButton{Text: "⬅️ Предыдущая", CallbackData: fmt.Sprintf("subjects_page:%d", page-1)})
		}

		// Показываем номер страницы
		paginationButtons = append(paginationButtons,
			models.InlineKeyboardButton{Text: fmt.Sprintf("📄 %d/%d", page+1, totalPages), CallbackData: "noop"})

		// Кнопка "Следующая" только если не последняя страница
		if page < totalPages-1 {
			paginationButtons = append(paginationButtons,
				models.InlineKeyboardButton{Text: "Следующая ➡️", CallbackData: fmt.Sprintf("subjects_page:%d", page+1)})
		}

		buttons = append(buttons, paginationButtons)
	}

	// Кнопка создать новый предмет
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "➕ Создать новый предмет", CallbackData: "create_first_subject"},
	})

	// Кнопка для быстрого перехода к расписанию
	buttons = append(buttons, []models.InlineKeyboardButton{
		{Text: "📅 Управление расписанием", CallbackData: "view_schedule"},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	return text, keyboard
}

// BuildDeleteSubjectConfirmScreen формирует экран подтверждения удаления предмета
func BuildDeleteSubjectConfirmScreen(subject *model.Subject, bookingsCount int) (string, *models.InlineKeyboardMarkup) {
	warningText := ""
	if bookingsCount > 0 {
		warningText = fmt.Sprintf("\n\n⚠️ **ВНИМАНИЕ!** У этого предмета есть %d активных бронирований.\n"+
			"Все студенты будут уведомлены об отмене.", bookingsCount)
	}

	text := fmt.Sprintf(
		"❓ Вы уверены, что хотите удалить предмет <b>%s</b>?\n\n"+
			"Это действие удалит:\n"+
			"• Сам предмет\n"+
			"• Все временные слоты\n"+
			"• Все связанные бронирования%s",
		subject.Name,
		warningText,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Да, удалить", CallbackData: fmt.Sprintf("confirm_delete:%d", subject.ID)},
			},
			{
				{Text: "⬅️ Назад", CallbackData: fmt.Sprintf("view_subject:%d", subject.ID)},
			},
		},
	}

	return text, keyboard
}

// ========================
// Student Screens
// ========================

// BuildStudentSubjectDetailsScreen формирует экран деталей subject для студента
func BuildStudentSubjectDetailsScreen(subject *model.Subject, teacherName string) (string, *models.InlineKeyboardMarkup) {
	approvalText := ""
	if subject.RequiresBookingApproval {
		approvalText = "\n⏳ Требуется одобрение учителя"
	}

	text := fmt.Sprintf(
		"📚 **%s**\n\n"+
			"👤 Преподаватель: %s\n"+
			"💰 Цена: %.2f ₽\n"+
			"⏱ Длительность: %d мин\n\n"+
			"📝 Описание:\n%s%s",
		subject.Name,
		teacherName,
		float64(subject.Price)/100,
		subject.Duration,
		subject.Description,
		approvalText,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📅 Посмотреть расписание", CallbackData: fmt.Sprintf("view_schedule_subject:%d", subject.ID)},
			},
			{
				{Text: "⬅️ К списку предметов", CallbackData: "book_another"},
			},
		},
	}

	return text, keyboard
}

// BuildBookingSuccessScreen формирует экран успешного бронирования
func BuildBookingSuccessScreen(bookingID int64, slotID int64, isPending bool) (string, *models.InlineKeyboardMarkup) {
	statusText := "Подтверждена ✅"
	additionalInfo := "Учитель получил уведомление о вашей записи."

	if isPending {
		statusText = "Ожидает одобрения ⏳"
		additionalInfo = "Учитель получил запрос на одобрение.\nВы получите уведомление после проверки."
	}

	text := fmt.Sprintf(
		"✅ Запись успешно создана!\n\n"+
			"📝 Запись #%d\n"+
			"📅 Статус: %s\n"+
			"📍 ID слота: %d\n\n"+
			"%s\n"+
			"Детали занятия будут доступны в /mybookings",
		bookingID,
		statusText,
		slotID,
		additionalInfo,
	)

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "📅 Мои записи", CallbackData: "back_to_main"}},
			{{Text: "➕ Записаться ещё", CallbackData: "book_another"}},
		},
	}

	return text, keyboard
}

// BuildEmptyBookingsScreen формирует экран для пустого списка бронирований
func BuildEmptyBookingsScreen() (string, *models.InlineKeyboardMarkup) {
	text := "📅 У вас пока нет записей на занятия.\n\nПосмотрите доступные предметы и запишитесь!"

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📚 Посмотреть предметы", CallbackData: "book_another"},
			},
		},
	}

	return text, keyboard
}

// BuildSubjectCategoriesScreen формирует экран выбора категории предметов
func BuildSubjectCategoriesScreen() (string, *models.InlineKeyboardMarkup) {
	text := "📚 *Предметы и учителя*\n\n" +
		"Выберите категорию:\n\n" +
		"🎓 *Мои учителя* - учителя, к которым у вас есть доступ\n" +
		"🌍 *Публичные учителя* - доступны всем студентам\n" +
		"🔍 *Найти учителя* - по коду приглашения или заявке\n" +
		"📋 *Мои заявки* - статус ваших запросов на доступ"

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "🎓 Мои учителя", CallbackData: "my_teachers"}},
			{{Text: "🌍 Публичные учителя", CallbackData: "public_teachers"}},
			{{Text: "🔍 Найти учителя", CallbackData: "find_teacher"}},
			{{Text: "📋 Мои заявки", CallbackData: "my_requests"}},
		},
	}

	return text, keyboard
}
