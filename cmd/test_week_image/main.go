package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/model"
)

func main() {
	// Создаем тестовые данные
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Начинаем с понедельника текущей недели
	for startDate.Weekday() != time.Monday {
		startDate = startDate.AddDate(0, 0, -1)
	}
	endDate := startDate.AddDate(0, 0, 6) // неделя (7 дней)

	// Создаем тестовые слоты
	slots := []*model.ScheduleSlot{
		// Понедельник
		{
			ID:        1,
			SubjectID: 1,
			StartTime: startDate.Add(9 * time.Hour),  // 09:00
			EndTime:   startDate.Add(10 * time.Hour), // 10:00
			Status:    model.SlotStatusFree,
			StudentID: nil,
		},
		{
			ID:        2,
			SubjectID: 1,
			StartTime: startDate.Add(14 * time.Hour), // 14:00
			EndTime:   startDate.Add(15 * time.Hour), // 15:00
			Status:    model.SlotStatusBooked,
			StudentID: intPtr(100),
		},
		// Вторник
		{
			ID:        3,
			SubjectID: 1,
			StartTime: startDate.AddDate(0, 0, 1).Add(10 * time.Hour), // Вторник 10:00
			EndTime:   startDate.AddDate(0, 0, 1).Add(11 * time.Hour), // Вторник 11:00
			Status:    model.SlotStatusFree,
			StudentID: nil,
		},
		{
			ID:        4,
			SubjectID: 1,
			StartTime: startDate.AddDate(0, 0, 1).Add(16 * time.Hour), // Вторник 16:00
			EndTime:   startDate.AddDate(0, 0, 1).Add(17 * time.Hour), // Вторник 17:00
			Status:    model.SlotStatusCanceled,
			StudentID: nil,
		},
		// Среда
		{
			ID:        5,
			SubjectID: 1,
			StartTime: startDate.AddDate(0, 0, 2).Add(9 * time.Hour),  // Среда 09:00
			EndTime:   startDate.AddDate(0, 0, 2).Add(10 * time.Hour), // Среда 10:00
			Status:    model.SlotStatusBooked,
			StudentID: intPtr(200),
		},
		{
			ID:        6,
			SubjectID: 1,
			StartTime: startDate.AddDate(0, 0, 2).Add(15 * time.Hour), // Среда 15:00
			EndTime:   startDate.AddDate(0, 0, 2).Add(16 * time.Hour), // Среда 16:00
			Status:    model.SlotStatusFree,
			StudentID: nil,
		},
		// Пятница
		{
			ID:        7,
			SubjectID: 1,
			StartTime: startDate.AddDate(0, 0, 4).Add(11 * time.Hour), // Пятница 11:00
			EndTime:   startDate.AddDate(0, 0, 4).Add(12 * time.Hour), // Пятница 12:00
			Status:    model.SlotStatusFree,
			StudentID: nil,
		},
		{
			ID:        8,
			SubjectID: 1,
			StartTime: startDate.AddDate(0, 0, 4).Add(13 * time.Hour), // Пятница 13:00
			EndTime:   startDate.AddDate(0, 0, 4).Add(14 * time.Hour), // Пятница 14:00
			Status:    model.SlotStatusBooked,
			StudentID: nil, // занят преподавателем без студента
		},
	}

	// Генерируем изображение
	imageData, err := common.GenerateWeekImage(startDate, endDate, slots, 1)
	if err != nil {
		fmt.Printf("Ошибка генерации изображения: %v\n", err)
		os.Exit(1)
	}

	// Сохраняем в файл
	filename := "week.png"
	err = os.WriteFile(filename, imageData, 0644)
	if err != nil {
		fmt.Printf("Ошибка сохранения файла: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Изображение успешно сохранено в %s\n", filename)
	fmt.Printf("📅 Период: %s - %s\n", startDate.Format("02.01.2006"), endDate.Format("02.01.2006"))
	fmt.Printf("📊 Слотов: %d\n", len(slots))
}

func intPtr(i int64) *int64 {
	return &i
}
