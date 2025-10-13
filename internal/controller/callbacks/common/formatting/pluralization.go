package formatting

// PluralizeSchedules возвращает правильное склонение слова "расписание"
func PluralizeSchedules(count int) string {
	if count%10 == 1 && count%100 != 11 {
		return "расписание"
	}
	if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return "расписания"
	}
	return "расписаний"
}

// PluralizeSlots возвращает правильное склонение слова "слот"
func PluralizeSlots(count int) string {
	if count%10 == 1 && count%100 != 11 {
		return "слот"
	}
	if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return "слота"
	}
	return "слотов"
}

// PluralizeWeeks возвращает правильное склонение слова "неделя"
func PluralizeWeeks(count int) string {
	if count == 1 {
		return "неделю"
	}
	if count >= 2 && count <= 4 {
		return "недели"
	}
	return "недель"
}

// PluralizeStudents возвращает правильное склонение слова "студент"
func PluralizeStudents(count int) string {
	if count%10 == 1 && count%100 != 11 {
		return "студент"
	}
	if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return "студента"
	}
	return "студентов"
}

// PluralizeBookings возвращает правильное склонение слова "запись"
func PluralizeBookings(count int) string {
	if count%10 == 1 && count%100 != 11 {
		return "запись"
	}
	if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return "записи"
	}
	return "записей"
}
