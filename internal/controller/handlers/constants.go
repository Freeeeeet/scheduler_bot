package handlers

// Константы валидации для создания предмета
const (
	// Название предмета
	SubjectNameMinLength = 3
	SubjectNameMaxLength = 100

	// Описание предмета
	SubjectDescriptionMinLength = 5
	SubjectDescriptionMaxLength = 500

	// Цена предмета (в рублях)
	SubjectMaxPrice = 1_000_000

	// Длительность занятия (в минутах)
	SubjectMinDuration = 15  // 15 минут
	SubjectMaxDuration = 480 // 8 часов
)
