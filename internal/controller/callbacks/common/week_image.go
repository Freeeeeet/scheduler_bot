package common

import (
	"bytes"
	_ "embed"
	"image/color"
	"strconv"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

// FontStyle определяет стиль шрифта
type FontStyle string

const (
	FontStyleDefault  FontStyle = "" // Regular
	FontStyleMedium   FontStyle = "medium"
	FontStyleItalic   FontStyle = "italic"
	FontStyleBold     FontStyle = "bold"
	FontStyleSemiBold FontStyle = "semi-bold"
	FontStyleLight    FontStyle = "light"
)

// Константы размеров и отступов
const (
	imageWidth       = 1400
	imageHeight      = 900
	headerHeight     = 100
	leftLabelsWidth  = 80
	legendWidth      = 120
	dayPaddingX      = 8
	minSlotHeight    = 8.0
	slotBorderRadius = 6.0
	shadowOffset     = 3.0
	totalDaysInWeek  = 7
	hourPaddingTop   = 2
	hourPaddingBot   = 2
	defaultMinHour   = 0
	defaultMaxHour   = 23
)

// Константы шрифтов
const (
	titleFontSize      = 25.0
	dayFontSize        = 27.0
	hourLabelFontSize  = 18.0
	slotTimeFontSize   = 17.0
	legendItemFontSize = 12.0
)

// Цветовая схема
var (
	bgColor          = color.RGBA{245, 246, 248, 255}
	textColor        = color.RGBA{80, 85, 90, 220}
	hourLabelColor   = color.RGBA{110, 115, 120, 200}
	hourLineColor    = color.NRGBA{150, 150, 150, 255}
	todayBgColor     = color.NRGBA{255, 99, 71, 125}
	evenDayColor     = color.NRGBA{240, 240, 240, 255}
	oddDayColor      = color.NRGBA{220, 220, 220, 255}
	currentTimeColor = color.NRGBA{255, 80, 80, 200}

	slotFreeColor       = color.RGBA{133, 193, 85, 220}
	slotBookedColor     = color.RGBA{255, 182, 193, 255} // Светло-розовый для забронированных
	slotCanceledColor   = color.RGBA{158, 158, 158, 200}
	slotDefaultColor    = color.RGBA{220, 220, 220, 200}
	slotTextColor       = color.RGBA{20, 24, 28, 230}  // Темный текст для светлых слотов
	slotBookedTextColor = color.RGBA{120, 40, 50, 255} // Темно-красный текст для забронированных
	slotShadowColor     = color.RGBA{0, 0, 0, 20}

	legendTextColor = color.RGBA{90, 95, 100, 220}
	legendItemColor = color.RGBA{70, 74, 78, 220}
)

// weekBounds содержит границы недели
type weekBounds struct {
	start time.Time
	end   time.Time
}

// hourRange содержит диапазон часов для отображения
type hourRange struct {
	start int
	end   int
	total int
}

//go:embed fonts/LibertinusSerif-Regular.ttf
var libertinusRegularFontData []byte

//go:embed fonts/LibertinusSerif-Bold.ttf
var libertinusBoldFontData []byte

//go:embed fonts/LibertinusSerif-Italic.ttf
var libertinusItalicFontData []byte

//go:embed fonts/LibertinusSerif-BoldItalic.ttf
var libertinusBoldItalicFontData []byte

//go:embed fonts/LibertinusSerif-SemiBold.ttf
var libertinusSemiBoldFontData []byte

//go:embed fonts/LibertinusSerif-SemiBoldItalic.ttf
var libertinusSemiBoldItalicFontData []byte

var cachedFonts = make(map[FontStyle]*opentype.Font)

// loadFont загружает шрифт указанного стиля или использует basicfont как fallback
func loadFont(dc *gg.Context, size float64, style ...FontStyle) {
	var fontStyle FontStyle = FontStyleDefault
	if len(style) > 0 {
		fontStyle = style[0]
	}

	// Определяем какие данные шрифта использовать
	var fontData []byte
	switch fontStyle {
	case FontStyleMedium:
		// Medium нет в LibertinusSerif, используем SemiBold
		fontData = libertinusSemiBoldFontData
	case FontStyleItalic:
		fontData = libertinusItalicFontData
	case FontStyleBold:
		fontData = libertinusBoldFontData
	case FontStyleSemiBold:
		fontData = libertinusSemiBoldFontData
	case FontStyleLight:
		// Light нет в LibertinusSerif, используем Regular
		fontData = libertinusRegularFontData
	default:
		fontData = libertinusRegularFontData
	}

	// Если нет данных для выбранного стиля, используем Regular
	if len(fontData) == 0 {
		fontData = libertinusRegularFontData
	}

	if len(fontData) > 0 {
		// Кешируем парсинг шрифта
		cachedFont, ok := cachedFonts[fontStyle]
		if !ok || cachedFont == nil {
			parsedFont, err := opentype.Parse(fontData)
			if err != nil {
				dc.SetFontFace(basicfont.Face7x13)
				return
			}
			cachedFonts[fontStyle] = parsedFont
			cachedFont = parsedFont
		}

		// Создаем face с нужным размером
		face, err := opentype.NewFace(cachedFont, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err == nil {
			dc.SetFontFace(face)
			return
		}
	}
	// fallback к встроенному шрифту
	dc.SetFontFace(basicfont.Face7x13)
}

// GenerateWeekImage генерирует изображение недели с отображением слотов
// studentNames - map studentID -> имя студента для отображения на слотах
func GenerateWeekImage(startDate, endDate time.Time, slots []*model.ScheduleSlot, subjectID int64, studentNames map[int64]string) ([]byte, error) {
	week := normalizeToWeekBounds(startDate)
	today := normalizeToDay(time.Now())
	shouldHighlightToday := isTodayInWeek(today, week)

	slotsByDay := groupSlotsByDay(slots, subjectID)
	hours := calculateHourRange(slots, subjectID)

	dc := createCanvas()
	dayWidth := (imageWidth - leftLabelsWidth - legendWidth) / totalDaysInWeek
	dayHeight := imageHeight - headerHeight
	cellHeight := float64(dayHeight) / float64(hours.total)

	drawHeader(dc, week)
	drawHourLabels(dc, hours, cellHeight)
	drawDaysAndSlots(dc, week, today, shouldHighlightToday, slotsByDay, hours, dayWidth, dayHeight, cellHeight, studentNames)
	drawCurrentTimeLine(dc, shouldHighlightToday, hours, cellHeight, dayWidth)
	drawLegend(dc, dayWidth)

	return encodeImage(dc)
}

// normalizeToWeekBounds нормализует дату к границам недели (Пн-Вс)
func normalizeToWeekBounds(date time.Time) weekBounds {
	normalized := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	daysSinceMonday := int(normalized.Weekday()) - 1
	if normalized.Weekday() == time.Sunday {
		daysSinceMonday = 6
	}

	start := normalized.AddDate(0, 0, -daysSinceMonday)
	end := start.AddDate(0, 0, 6)

	return weekBounds{start: start, end: end}
}

// normalizeToDay нормализует время к началу дня
func normalizeToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// isTodayInWeek проверяет, попадает ли сегодня в отображаемую неделю
func isTodayInWeek(today time.Time, week weekBounds) bool {
	return !today.Before(week.start) && !today.After(week.end)
}

// groupSlotsByDay группирует слоты по дням
func groupSlotsByDay(slots []*model.ScheduleSlot, subjectID int64) map[string][]*model.ScheduleSlot {
	slotsByDay := make(map[string][]*model.ScheduleSlot)
	for _, slot := range slots {
		if slot.SubjectID == subjectID {
			dateKey := slot.StartTime.Format("2006-01-02")
			slotsByDay[dateKey] = append(slotsByDay[dateKey], slot)
		}
	}
	return slotsByDay
}

// calculateHourRange определяет диапазон часов для отображения
func calculateHourRange(slots []*model.ScheduleSlot, subjectID int64) hourRange {
	minHour := 24
	maxHour := 0

	for _, slot := range slots {
		if slot.SubjectID == subjectID {
			startH := slot.StartTime.Hour()
			endH := slot.EndTime.Hour()
			if slot.EndTime.Minute() > 0 {
				endH++
			}
			if startH < minHour {
				minHour = startH
			}
			if endH > maxHour {
				maxHour = endH
			}
		}
	}

	if minHour == 24 {
		minHour = defaultMinHour
		maxHour = defaultMaxHour
	}

	startHour := minHour - hourPaddingTop
	endHour := maxHour + hourPaddingBot
	if startHour < 0 {
		startHour = 0
	}
	if endHour > 23 {
		endHour = 23
	}

	return hourRange{
		start: startHour,
		end:   endHour,
		total: endHour - startHour + 1,
	}
}

// createCanvas создает новый контекст рисования с фоном
func createCanvas() *gg.Context {
	dc := gg.NewContext(imageWidth, imageHeight)
	dc.SetColor(bgColor)
	dc.Clear()
	return dc
}

// drawHeader рисует заголовок с названием месяца
func drawHeader(dc *gg.Context, week weekBounds) {
	startMonth := week.start.Month()
	endMonth := week.end.Month()

	var title string
	if startMonth == endMonth {
		title = getMonthNameRussian(startMonth)
	} else {
		title = getMonthNameRussian(startMonth) + " - " + getMonthNameRussian(endMonth)
	}

	loadFont(dc, titleFontSize, FontStyleBold)
	dc.SetColor(textColor)
	w, h := dc.MeasureString(title)
	dc.DrawStringAnchored(title, w/2, float64(headerHeight)/8+h/2, 0, 0)
}

// drawHourLabels рисует колонку с часами слева
func drawHourLabels(dc *gg.Context, hours hourRange, cellHeight float64) {
	loadFont(dc, hourLabelFontSize, FontStyleMedium)
	dc.SetColor(hourLabelColor)

	for hIdx := 0; hIdx < hours.total; hIdx++ {
		actualHour := hours.start + hIdx
		y := float64(headerHeight) + float64(hIdx)*cellHeight
		timeLabel := formatHourLabel(actualHour)
		dc.DrawStringAnchored(timeLabel, float64(leftLabelsWidth)-10, y, 1, 0.5)
	}
}

// drawDaysAndSlots рисует все дни недели со слотами
func drawDaysAndSlots(dc *gg.Context, week weekBounds, today time.Time, shouldHighlightToday bool,
	slotsByDay map[string][]*model.ScheduleSlot, hours hourRange, dayWidth, dayHeight int, cellHeight float64, studentNames map[int64]string) {

	currentDate := week.start

	for dayIndex := 0; dayIndex < totalDaysInWeek; dayIndex++ {
		x := float64(leftLabelsWidth + dayIndex*dayWidth)
		y := float64(headerHeight)

		isToday := shouldHighlightToday && isSameDay(currentDate, today)

		drawDayBackground(dc, x, y, dayWidth, dayHeight, dayIndex, isToday)
		drawDayHeader(dc, currentDate, x, y, dayWidth)
		drawHourLines(dc, x, y, dayWidth, hours, cellHeight)
		drawSlotsForDay(dc, currentDate, slotsByDay, x, y, dayWidth, hours, cellHeight, studentNames)

		currentDate = currentDate.AddDate(0, 0, 1)
	}
}

// isSameDay проверяет, являются ли две даты одним днем
func isSameDay(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() &&
		date1.Month() == date2.Month() &&
		date1.Day() == date2.Day()
}

// drawDayBackground рисует фон дня
func drawDayBackground(dc *gg.Context, x, y float64, dayWidth, dayHeight, dayIndex int, isToday bool) {
	if isToday {
		dc.SetColor(todayBgColor)
	} else if dayIndex%2 == 0 {
		dc.SetColor(evenDayColor)
	} else {
		dc.SetColor(oddDayColor)
	}
	dc.DrawRectangle(x, y, float64(dayWidth), float64(dayHeight))
	dc.Fill()
}

// drawDayHeader рисует название дня недели и дату
func drawDayHeader(dc *gg.Context, date time.Time, x, y float64, dayWidth int) {
	weekdayStr := getWeekdayShort(date.Weekday())
	dateStr := date.Format("02.01")

	loadFont(dc, dayFontSize, FontStyleBold)
	dc.SetColor(textColor)
	dc.DrawStringAnchored(dateStr, x+float64(dayWidth)/2, y, 0.5, -1)
	dc.DrawStringAnchored(weekdayStr, x+float64(dayWidth)/2, y, 0.5, -0.2)
}

// drawHourLines рисует горизонтальные линии часов
func drawHourLines(dc *gg.Context, x, y float64, dayWidth int, hours hourRange, cellHeight float64) {
	dc.SetLineWidth(0.3)
	dc.SetColor(hourLineColor)

	for hIdx := 0; hIdx <= hours.total; hIdx++ {
		hy := y + float64(hIdx)*cellHeight
		dc.DrawLine(x, hy, x+float64(dayWidth), hy)
		dc.Stroke()
	}
}

// drawSlotsForDay рисует все слоты для указанного дня
func drawSlotsForDay(dc *gg.Context, date time.Time, slotsByDay map[string][]*model.ScheduleSlot,
	x, y float64, dayWidth int, hours hourRange, cellHeight float64, studentNames map[int64]string) {

	dateKey := date.Format("2006-01-02")
	for _, slot := range slotsByDay[dateKey] {
		drawSlot(dc, slot, x, y, dayWidth, hours, cellHeight, studentNames)
	}
}

// drawSlot рисует один слот
func drawSlot(dc *gg.Context, slot *model.ScheduleSlot, x, y float64, dayWidth int, hours hourRange, cellHeight float64, studentNames map[int64]string) {
	slotStartHour := float64(slot.StartTime.Hour()) + float64(slot.StartTime.Minute())/60.0
	slotEndHour := float64(slot.EndTime.Hour()) + float64(slot.EndTime.Minute())/60.0

	slotY := y + (slotStartHour-float64(hours.start))*cellHeight
	slotHeight := (slotEndHour - slotStartHour) * cellHeight
	if slotHeight < minSlotHeight {
		slotHeight = minSlotHeight
	}

	fillColor := getSlotColor(slot.Status)
	slotWidth := float64(dayWidth) - float64(dayPaddingX*2)

	// Тень
	dc.SetColor(slotShadowColor)
	dc.DrawRoundedRectangle(x+dayPaddingX+shadowOffset, slotY+2+shadowOffset, slotWidth, slotHeight-4, slotBorderRadius)
	dc.Fill()

	// Основной слот
	dc.SetColor(fillColor)
	dc.DrawRoundedRectangle(x+float64(dayPaddingX), slotY+2, slotWidth, slotHeight-4, slotBorderRadius)
	dc.Fill()

	// Рамка
	borderColor := darkenColor(fillColor, 0.8)
	dc.SetColor(borderColor)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(x+float64(dayPaddingX), slotY+2, slotWidth, slotHeight-4, slotBorderRadius)
	dc.Stroke()

	// Выбираем цвет текста в зависимости от статуса слота
	textColor := slotTextColor
	if slot.Status == model.SlotStatusBooked {
		textColor = slotBookedTextColor
	}

	// Текст времени
	loadFont(dc, slotTimeFontSize, FontStyleMedium)
	dc.SetColor(textColor)
	txtX := x + float64(dayPaddingX) + 8
	txtY := slotY + 8 + 10
	timeText := slot.StartTime.Format("15:04")
	dc.DrawStringAnchored(timeText, txtX, txtY, 0, 0)

	// Добавляем комментарий или имя студента, если есть
	additionalText := ""
	if slot.Comment != nil && *slot.Comment != "" {
		// Приоритет комментарию
		additionalText = *slot.Comment
	} else if slot.StudentID != nil && slot.Status == model.SlotStatusBooked {
		// Если нет комментария, но есть студент
		if name, ok := studentNames[*slot.StudentID]; ok && name != "" {
			additionalText = name
		}
	}

	// Отображаем дополнительный текст (комментарий или имя) если есть место
	if additionalText != "" && slotHeight > 25 {
		// Ограничиваем длину текста для отображения
		maxLen := 20
		if len(additionalText) > maxLen {
			additionalText = additionalText[:maxLen-3] + "..."
		}
		// Используем меньший шрифт для дополнительного текста
		loadFont(dc, slotTimeFontSize-2, FontStyleMedium)
		dc.SetColor(textColor)
		dc.DrawStringAnchored(additionalText, txtX, txtY+16, 0, 0)
	}
}

// getSlotColor возвращает цвет слота по его статусу
func getSlotColor(status model.SlotStatus) color.RGBA {
	switch status {
	case model.SlotStatusFree:
		return slotFreeColor
	case model.SlotStatusBooked:
		return slotBookedColor
	case model.SlotStatusCanceled:
		return slotCanceledColor
	default:
		return slotDefaultColor
	}
}

// darkenColor затемняет цвет на указанный множитель
func darkenColor(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * factor),
		G: uint8(float64(c.G) * factor),
		B: uint8(float64(c.B) * factor),
		A: c.A,
	}
}

// drawCurrentTimeLine рисует красную линию текущего времени
func drawCurrentTimeLine(dc *gg.Context, shouldHighlight bool, hours hourRange, cellHeight float64, dayWidth int) {
	if !shouldHighlight {
		return
	}

	now := time.Now()
	currentHour := float64(now.Hour()) + float64(now.Minute())/60.0

	if currentHour < float64(hours.start) || currentHour > float64(hours.end) {
		return
	}

	currentTimeY := float64(headerHeight) + (currentHour-float64(hours.start))*cellHeight
	dc.SetColor(currentTimeColor)
	dc.SetLineWidth(2.0)
	dc.DrawLine(float64(leftLabelsWidth), currentTimeY, float64(leftLabelsWidth+totalDaysInWeek*dayWidth), currentTimeY)
	dc.Stroke()
}

// drawLegend рисует легенду справа
func drawLegend(dc *gg.Context, dayWidth int) {
	legendX := float64(leftLabelsWidth + totalDaysInWeek*dayWidth + 10)
	legendY := float64(imageHeight) - 100.0

	dc.SetColor(legendTextColor)

	legendItems := []struct {
		Label string
		Clr   color.Color
	}{
		{"Свободно", slotFreeColor},
		{"Забронировано", slotBookedColor},
		{"Отменено", slotCanceledColor},
	}

	boxW := 20.0
	boxH := 14.0
	liX := legendX
	liY := legendY + 22

	for _, item := range legendItems {
		dc.SetColor(item.Clr)
		dc.DrawRoundedRectangle(liX, liY, boxW, boxH, 3)
		dc.Fill()

		loadFont(dc, legendItemFontSize)
		dc.SetColor(legendItemColor)
		dc.DrawStringAnchored(item.Label, liX+boxW+8, liY+boxH/2+1, 0, 0.2)
		liY += boxH + 14
	}
}

// encodeImage кодирует изображение в PNG
func encodeImage(dc *gg.Context) ([]byte, error) {
	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// формат числа с двумя цифрами
func formatTwoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func formatHourLabel(h int) string {
	return formatTwoDigits(h) + ":00"
}

// короткие дни недели
func getWeekdayShort(weekday time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Monday:    "Пн",
		time.Tuesday:   "Вт",
		time.Wednesday: "Ср",
		time.Thursday:  "Чт",
		time.Friday:    "Пт",
		time.Saturday:  "Сб",
		time.Sunday:    "Вс",
	}
	return weekdays[weekday]
}

// названия месяцев на русском
func getMonthNameRussian(month time.Month) string {
	months := map[time.Month]string{
		time.January:   "Январь",
		time.February:  "Февраль",
		time.March:     "Март",
		time.April:     "Апрель",
		time.May:       "Май",
		time.June:      "Июнь",
		time.July:      "Июль",
		time.August:    "Август",
		time.September: "Сентябрь",
		time.October:   "Октябрь",
		time.November:  "Ноябрь",
		time.December:  "Декабрь",
	}
	return months[month]
}
