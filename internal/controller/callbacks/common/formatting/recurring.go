package formatting

import (
	"fmt"
	"sort"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
)

// RecurringScheduleGroup представляет группу recurring schedules с общими параметрами
type RecurringScheduleGroup struct {
	Weekdays []int
	MinTime  string // "07:00"
	MaxTime  string // "17:00"
	IDs      []int64
}

// GroupRecurringSchedules группирует recurring schedules по дням недели и времени
// Возвращает группы, отсортированные по дню недели и времени
func GroupRecurringSchedules(schedules []*model.RecurringSchedule) []*RecurringScheduleGroup {
	if len(schedules) == 0 {
		return nil
	}

	// Создаем ключ из отсортированных дней недели
	weekdayGroupKey := func(weekdays []int) string {
		sorted := make([]int, len(weekdays))
		copy(sorted, weekdays)
		sort.Ints(sorted)
		key := ""
		for _, wd := range sorted {
			key += fmt.Sprintf("%d,", wd)
		}
		return key
	}

	// Собираем расписания по дням недели и времени
	weekdayToSchedules := make(map[string][]*model.RecurringSchedule)

	for _, rs := range schedules {
		if !rs.IsActive {
			continue
		}

		// Находим все расписания с тем же временем
		var sameTimeSchedules []*model.RecurringSchedule
		for _, rs2 := range schedules {
			if !rs2.IsActive {
				continue
			}
			if rs2.StartHour == rs.StartHour && rs2.StartMinute == rs.StartMinute {
				sameTimeSchedules = append(sameTimeSchedules, rs2)
			}
		}

		// Получаем дни недели для этой группы времени
		var weekdays []int
		for _, s := range sameTimeSchedules {
			weekdays = append(weekdays, s.Weekday)
		}
		sort.Ints(weekdays)
		key := weekdayGroupKey(weekdays)

		// Добавляем только если еще не добавили
		if _, exists := weekdayToSchedules[key]; !exists {
			weekdayToSchedules[key] = sameTimeSchedules
		}
	}

	// Создаем финальные группы с диапазонами времени
	var groups []*RecurringScheduleGroup
	for _, schedules := range weekdayToSchedules {
		if len(schedules) == 0 {
			continue
		}

		var weekdays []int
		var ids []int64
		minTime := "23:59"
		maxTime := "00:00"

		for _, rs := range schedules {
			timeStr := fmt.Sprintf("%02d:%02d", rs.StartHour, rs.StartMinute)
			if timeStr < minTime {
				minTime = timeStr
			}

			// Рассчитываем время окончания (начало + длительность)
			endTime := time.Date(2000, 1, 1, rs.StartHour, rs.StartMinute, 0, 0, time.UTC).
				Add(time.Duration(rs.DurationMinutes) * time.Minute)
			endTimeStr := endTime.Format("15:04")
			if endTimeStr > maxTime {
				maxTime = endTimeStr
			}

			// Собираем уникальные дни недели
			found := false
			for _, wd := range weekdays {
				if wd == rs.Weekday {
					found = true
					break
				}
			}
			if !found {
				weekdays = append(weekdays, rs.Weekday)
			}
			ids = append(ids, rs.ID)
		}

		sort.Ints(weekdays)
		groups = append(groups, &RecurringScheduleGroup{
			Weekdays: weekdays,
			MinTime:  minTime,
			MaxTime:  maxTime,
			IDs:      ids,
		})
	}

	// Сортируем группы по первому дню недели, затем по времени
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Weekdays) > 0 && len(groups[j].Weekdays) > 0 {
			if groups[i].Weekdays[0] != groups[j].Weekdays[0] {
				return groups[i].Weekdays[0] < groups[j].Weekdays[0]
			}
		}
		return groups[i].MinTime < groups[j].MinTime
	})

	return groups
}

// FormatRecurringGroupDisplay форматирует отображение группы recurring schedules
// Например: "Пн-Пт 09:00-18:00" или "Ср 14:00"
func FormatRecurringGroupDisplay(group *RecurringScheduleGroup) string {
	weekdaysStr := FormatWeekdayRange(group.Weekdays)
	timeRange := fmt.Sprintf("%s-%s", group.MinTime, group.MaxTime)

	// Если начало и конец совпадают (один слот), показываем только время
	if group.MinTime == group.MaxTime {
		timeRange = group.MinTime
	}

	return fmt.Sprintf("%s %s", weekdaysStr, timeRange)
}
