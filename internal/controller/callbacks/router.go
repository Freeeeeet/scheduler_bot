package callbacks

import (
	"context"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/student"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/teacher"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// ========================
// Callback Data Patterns
// ========================
// These constants define the callback data formats used throughout the bot

// Common callbacks
const (
	BackToMain  = "back_to_main"
	BookAnother = "book_another"
)

// Teacher callbacks - becoming a teacher and subject management
const (
	BecomeTeacher       = "become_teacher"
	CancelBecomeTeacher = "cancel_become_teacher"

	CreateFirstSubject = "create_first_subject"
	SkipFirstSubject   = "skip_first_subject"

	CreateSubjectApprovalYes = "create_subject_approval_yes"
	CreateSubjectApprovalNo  = "create_subject_approval_no"

	ViewSubject   = "view_subject:"   // view_subject:123
	EditSubject   = "edit_subject:"   // edit_subject:123
	ToggleSubject = "toggle_subject:" // toggle_subject:123
	DeleteSubject = "delete_subject:" // delete_subject:123
	ConfirmDelete = "confirm_delete:" // confirm_delete:123

	// Edit subject fields
	EditFieldName      = "edit_field_name:"      // edit_field_name:123
	EditFieldDesc      = "edit_field_desc:"      // edit_field_desc:123
	EditFieldPrice     = "edit_field_price:"     // edit_field_price:123
	EditFieldDuration  = "edit_field_duration:"  // edit_field_duration:123
	EditDurationCustom = "edit_duration_custom:" // edit_duration_custom:123
	ToggleApproval     = "toggle_approval:"      // toggle_approval:123
	SetDuration        = "set_duration:"         // set_duration:123:60 (ID:minutes)

	ViewSchedule        = "view_schedule"
	ViewScheduleSubject = "view_schedule_subject:" // view_schedule_subject:subject_id
	AddSlots            = "add_slots"
	CreateSlots         = "create_slots:" // create_slots:subject_id
	SetWeekday          = "set_weekday:"  // set_weekday:subject_id:weekday
	SetTime             = "set_time:"     // set_time:subject_id:weekday:hour
	ManualBook          = "manual_book"   // Teacher manually books a student
)

// Student callbacks - booking and cancellation
const (
	BookLesson    = "book_lesson:"    // book_lesson:slot_id
	CancelBooking = "cancel_booking:" // cancel_booking:booking_id
	ConfirmCancel = "confirm_cancel:" // confirm_cancel:booking_id
)

// Booking approval system callbacks (for future implementation)
const (
	ApproveBooking = "approve_booking:" // approve_booking:booking_id
	RejectBooking  = "reject_booking:"  // reject_booking:booking_id
	ApproveCancel  = "approve_cancel:"  // approve_cancel:booking_id
	RejectCancel   = "reject_cancel:"   // reject_cancel:booking_id
)

// ========================
// Main Callback Router
// ========================

// Route распределяет callback query по соответствующим обработчикам
func Route(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, h *callbacktypes.Handler) {
	data := callback.Data

	h.Logger.Info("Routing callback",
		zap.String("data", data),
		zap.Int64("user_id", callback.From.ID),
		zap.String("user_name", callback.From.FirstName))

	// Route callback to appropriate handler
	switch {
	// ===== Common Navigation =====
	case data == BackToMain:
		common.HandleBackToMain(ctx, b, callback, h)
	case data == BookAnother:
		common.HandleBookAnother(ctx, b, callback, h)

	// ===== Teacher: Becoming a Teacher =====
	case data == BecomeTeacher:
		teacher.HandleBecomeTeacherConfirm(ctx, b, callback, h)
	case data == CancelBecomeTeacher:
		teacher.HandleBecomeTeacherCancel(ctx, b, callback, h)

	// ===== Teacher: Subject Management =====
	case data == CreateFirstSubject:
		teacher.HandleCreateFirstSubject(ctx, b, callback, h)
	case data == SkipFirstSubject:
		teacher.HandleSkipFirstSubject(ctx, b, callback, h)
	case data == CreateSubjectApprovalYes:
		teacher.HandleCreateSubjectApprovalYes(ctx, b, callback, h)
	case data == CreateSubjectApprovalNo:
		teacher.HandleCreateSubjectApprovalNo(ctx, b, callback, h)
	case strings.HasPrefix(data, "create_subject_set_duration:"):
		teacher.HandleCreateSubjectSetDuration(ctx, b, callback, h)
	case strings.HasPrefix(data, ViewSubject):
		// Проверяем, является ли пользователь учителем-владельцем этого предмета
		subjectID, err := common.ParseIDFromCallback(data)
		if err == nil {
			telegramID := callback.From.ID
			user, userErr := h.UserService.GetByTelegramID(ctx, telegramID)
			subject, subjectErr := h.TeacherService.GetSubjectByID(ctx, subjectID)

			// Если это учитель-владелец предмета, показываем админский интерфейс
			if userErr == nil && subjectErr == nil && user != nil && subject != nil && subject.TeacherID == user.ID {
				teacher.HandleViewSubject(ctx, b, callback, h)
			} else {
				// Иначе показываем студенческий интерфейс
				student.HandleViewSubjectDetails(ctx, b, callback, h)
			}
		} else {
			common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
		}
	case strings.HasPrefix(data, EditSubject):
		teacher.HandleEditSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldName):
		teacher.HandleEditFieldName(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldDesc):
		teacher.HandleEditFieldDesc(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldPrice):
		teacher.HandleEditFieldPrice(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldDuration):
		teacher.HandleEditFieldDuration(ctx, b, callback, h)
	case strings.HasPrefix(data, ToggleApproval):
		teacher.HandleToggleApproval(ctx, b, callback, h)
	case strings.HasPrefix(data, SetDuration):
		teacher.HandleSetDuration(ctx, b, callback, h)
	case strings.HasPrefix(data, EditDurationCustom):
		teacher.HandleEditDurationCustom(ctx, b, callback, h)
	case strings.HasPrefix(data, ToggleSubject):
		teacher.HandleToggleSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, DeleteSubject):
		teacher.HandleDeleteSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, ConfirmDelete):
		teacher.HandleConfirmDeleteSubject(ctx, b, callback, h)

	// ===== Teacher: Schedule Management =====
	case data == ViewSchedule:
		teacher.HandleViewSchedule(ctx, b, callback, h)
	case data == AddSlots:
		teacher.HandleAddSlots(ctx, b, callback, h)
	case strings.HasPrefix(data, CreateSlots):
		teacher.HandleCreateSlotsStart(ctx, b, callback, h)
	case strings.HasPrefix(data, "slot_mode:"):
		teacher.HandleSlotMode(ctx, b, callback, h)
	case strings.HasPrefix(data, "single_day:"):
		teacher.HandleSingleDay(ctx, b, callback, h)
	case strings.HasPrefix(data, "single_time:"):
		teacher.HandleSingleDayTime(ctx, b, callback, h)
	case strings.HasPrefix(data, "period_weeks:"):
		teacher.HandlePeriodWeeks(ctx, b, callback, h)
	case strings.HasPrefix(data, "period_weekday:"):
		teacher.HandlePeriodWeekday(ctx, b, callback, h)
	case strings.HasPrefix(data, "period_time:"):
		teacher.HandlePeriodTime(ctx, b, callback, h)
	case strings.HasPrefix(data, "workday_day:"):
		teacher.HandleWorkdayDay(ctx, b, callback, h)
	case strings.HasPrefix(data, SetWeekday):
		teacher.HandleSetWeekday(ctx, b, callback, h)
	case strings.HasPrefix(data, SetTime):
		teacher.HandleSetTime(ctx, b, callback, h)
	case data == ManualBook:
		teacher.HandleManualBook(ctx, b, callback, h)

	// ===== Student: Booking Lessons =====
	case strings.HasPrefix(data, ViewScheduleSubject):
		student.HandleViewScheduleSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, BookLesson):
		student.HandleBookLesson(ctx, b, callback, h)
	case strings.HasPrefix(data, CancelBooking):
		student.HandleCancelBooking(ctx, b, callback, h)
	case strings.HasPrefix(data, ConfirmCancel):
		student.HandleConfirmCancel(ctx, b, callback, h)

	// ===== Teacher: Booking Approval System =====
	case strings.HasPrefix(data, ApproveBooking):
		student.HandleApproveBooking(ctx, b, callback, h)
	case strings.HasPrefix(data, RejectBooking):
		student.HandleRejectBooking(ctx, b, callback, h)
	case strings.HasPrefix(data, ApproveCancel):
		student.HandleApproveCancel(ctx, b, callback, h)
	case strings.HasPrefix(data, RejectCancel):
		student.HandleRejectCancel(ctx, b, callback, h)

	// ===== Unknown Callback =====
	default:
		h.Logger.Warn("Unknown callback",
			zap.String("data", data),
			zap.Int64("user_id", callback.From.ID))
		common.AnswerCallback(ctx, b, callback.ID, "❌ Неизвестная команда")
	}

	h.Logger.Info("Callback routed successfully", zap.String("data", data))
}
