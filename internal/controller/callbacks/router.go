package callbacks

import (
	"context"
	"strings"

	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/callbacktypes"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/common"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/student"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/teacher"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/teacher/recurring"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/teacher/schedule"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/teacher/slots"
	"github.com/Freeeeeet/scheduler_bot/internal/controller/callbacks/teacher/subjects"
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
	case strings.HasPrefix(data, "back_to_subjects"):
		common.HandleBackToSubjects(ctx, b, callback, h)
	case data == "noop":
		// No operation - просто подтверждаем callback
		common.AnswerCallback(ctx, b, callback.ID, "")

	// ===== Teacher: Becoming a Teacher =====
	case data == BecomeTeacher:
		teacher.HandleBecomeTeacherConfirm(ctx, b, callback, h)
	case data == CancelBecomeTeacher:
		teacher.HandleBecomeTeacherCancel(ctx, b, callback, h)

	// ===== Teacher: Subject Management =====
	case strings.HasPrefix(data, "subjects_page:"):
		subjects.HandleSubjectsPage(ctx, b, callback, h)
	case data == CreateFirstSubject:
		subjects.HandleCreateFirstSubject(ctx, b, callback, h)
	case data == SkipFirstSubject:
		subjects.HandleSkipFirstSubject(ctx, b, callback, h)
	case data == CreateSubjectApprovalYes:
		subjects.HandleCreateSubjectApprovalYes(ctx, b, callback, h)
	case data == CreateSubjectApprovalNo:
		subjects.HandleCreateSubjectApprovalNo(ctx, b, callback, h)
	case strings.HasPrefix(data, "create_subject_set_duration:"):
		subjects.HandleCreateSubjectSetDuration(ctx, b, callback, h)
	case strings.HasPrefix(data, ViewSubject):
		// Проверяем, является ли пользователь учителем-владельцем этого предмета
		subjectID, err := common.ParseIDFromCallback(data)
		if err != nil {
			h.Logger.Error("Failed to parse subject ID in view_subject", zap.Error(err), zap.String("data", data))
			common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
			return
		}

		telegramID := callback.From.ID
		user, userErr := h.UserService.GetByTelegramID(ctx, telegramID)
		if userErr != nil {
			h.Logger.Error("Failed to get user in view_subject", zap.Error(userErr), zap.Int64("telegram_id", telegramID))
			common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Ошибка получения пользователя")
			return
		}

		subject, subjectErr := h.TeacherService.GetSubjectByID(ctx, subjectID)
		if subjectErr != nil {
			h.Logger.Error("Failed to get subject in view_subject", zap.Error(subjectErr), zap.Int64("subject_id", subjectID))
			common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Предмет не найден")
			return
		}

		// Если это учитель-владелец предмета, показываем админский интерфейс
		if user != nil && subject != nil && user.IsTeacher && subject.TeacherID == user.ID {
			h.Logger.Info("Showing teacher interface for subject",
				zap.Int64("user_id", user.ID),
				zap.Int64("subject_id", subjectID))
			subjects.HandleViewSubject(ctx, b, callback, h)
		} else {
			// Иначе показываем студенческий интерфейс
			h.Logger.Info("Showing student interface for subject",
				zap.Int64("user_id", user.ID),
				zap.Int64("subject_id", subjectID))
			student.HandleViewSubjectDetails(ctx, b, callback, h)
		}
	case strings.HasPrefix(data, EditSubject):
		subjects.HandleEditSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldName):
		subjects.HandleEditFieldName(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldDesc):
		subjects.HandleEditFieldDesc(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldPrice):
		subjects.HandleEditFieldPrice(ctx, b, callback, h)
	case strings.HasPrefix(data, EditFieldDuration):
		subjects.HandleEditFieldDuration(ctx, b, callback, h)
	case strings.HasPrefix(data, ToggleApproval):
		subjects.HandleToggleApproval(ctx, b, callback, h)
	case strings.HasPrefix(data, SetDuration):
		subjects.HandleSetDuration(ctx, b, callback, h)
	case strings.HasPrefix(data, EditDurationCustom):
		subjects.HandleEditDurationCustom(ctx, b, callback, h)
	case strings.HasPrefix(data, ToggleSubject):
		subjects.HandleToggleSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, DeleteSubject):
		subjects.HandleDeleteSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, ConfirmDelete):
		subjects.HandleConfirmDeleteSubject(ctx, b, callback, h)

	// ===== Teacher: Schedule Management =====
	case data == ViewSchedule:
		schedule.HandleViewSchedule(ctx, b, callback, h)
	case strings.HasPrefix(data, "subject_schedule:"):
		schedule.HandleViewSubjectSchedule(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_schedule_calendar:"):
		schedule.HandleViewScheduleCalendar(ctx, b, callback, h)
	case strings.HasPrefix(data, "schedule_calendar_page:"):
		schedule.HandleViewScheduleCalendarPage(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_schedule_day:"):
		schedule.HandleViewScheduleDay(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_schedule_weeks:"):
		schedule.HandleViewScheduleWeeks(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_schedule_week_day:"):
		schedule.HandleViewScheduleWeekDay(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_slot_details:"):
		schedule.HandleViewSlotDetails(ctx, b, callback, h)
	case strings.HasPrefix(data, "cancel_slot:"):
		schedule.HandleCancelSlot(ctx, b, callback, h)
	case strings.HasPrefix(data, "restore_slot:"):
		schedule.HandleRestoreSlot(ctx, b, callback, h)
	case strings.HasPrefix(data, "cancel_booking_from_slot:"):
		schedule.HandleCancelBookingFromSlot(ctx, b, callback, h)
	case strings.HasPrefix(data, "manage_temporary:"):
		schedule.HandleManageTemporary(ctx, b, callback, h)
	case data == "back_to_myschedule":
		common.HandleBackToMySchedule(ctx, b, callback, h)
	case strings.HasPrefix(data, "manage_recurring:"):
		recurring.HandleManageRecurring(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_recurring_group:"):
		recurring.HandleViewRecurringGroup(ctx, b, callback, h)
	case strings.HasPrefix(data, "delete_recurring_group:"):
		recurring.HandleDeleteRecurringGroup(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_all_slots:"):
		schedule.HandleViewAllSlots(ctx, b, callback, h)
	case strings.HasPrefix(data, "toggle_recurring:"):
		recurring.HandleToggleRecurring(ctx, b, callback, h)
	case strings.HasPrefix(data, "edit_recurring_menu:"):
		recurring.HandleEditRecurringMenu(ctx, b, callback, h)
	case strings.HasPrefix(data, "edit_recurring_days:"):
		recurring.HandleEditRecurringDays(ctx, b, callback, h)
	case strings.HasPrefix(data, "toggle_edit_weekday:"):
		recurring.HandleToggleEditWeekday(ctx, b, callback, h)
	case strings.HasPrefix(data, "save_recurring_days:"):
		recurring.HandleSaveRecurringDays(ctx, b, callback, h)
	case strings.HasPrefix(data, "edit_recurring_time:"):
		recurring.HandleEditRecurringTime(ctx, b, callback, h)
	case strings.HasPrefix(data, "recurring_edit_time_mode:"):
		recurring.HandleRecurringEditTimeMode(ctx, b, callback, h)
	case strings.HasPrefix(data, "recurring_edit_interval_start:"):
		recurring.HandleRecurringEditIntervalStart(ctx, b, callback, h)
	case strings.HasPrefix(data, "recurring_edit_interval_end:"):
		recurring.HandleRecurringEditIntervalEnd(ctx, b, callback, h)
	case strings.HasPrefix(data, "create_recurring_start:"):
		recurring.HandleCreateRecurringStart(ctx, b, callback, h)
	case strings.HasPrefix(data, "toggle_create_weekday:"):
		recurring.HandleToggleCreateWeekday(ctx, b, callback, h)
	case strings.HasPrefix(data, "create_recurring_continue:"):
		recurring.HandleCreateRecurringContinue(ctx, b, callback, h)
	case strings.HasPrefix(data, "recurring_time_mode:"):
		recurring.HandleRecurringTimeMode(ctx, b, callback, h)
	case strings.HasPrefix(data, "recurring_interval_start:"):
		recurring.HandleRecurringIntervalStart(ctx, b, callback, h)
	case strings.HasPrefix(data, "recurring_interval_end:"):
		recurring.HandleRecurringIntervalEnd(ctx, b, callback, h)
	case strings.HasPrefix(data, "toggle_time_slot:"):
		recurring.HandleToggleTimeSlot(ctx, b, callback, h)
	case strings.HasPrefix(data, "create_recurring_specific_confirm:"):
		recurring.HandleCreateRecurringSpecificConfirm(ctx, b, callback, h)
	case data == AddSlots:
		schedule.HandleAddSlots(ctx, b, callback, h)
	case strings.HasPrefix(data, CreateSlots):
		slots.HandleCreateSlotsStart(ctx, b, callback, h)
	case strings.HasPrefix(data, "slot_mode:"):
		slots.HandleSlotMode(ctx, b, callback, h)
	case strings.HasPrefix(data, "single_day_page:"):
		slots.HandleSingleDayPage(ctx, b, callback, h)
	case strings.HasPrefix(data, "single_day_date:"):
		slots.HandleSingleDayDate(ctx, b, callback, h)
	case strings.HasPrefix(data, "single_time_auto:"):
		slots.HandleSingleTimeAuto(ctx, b, callback, h)
	case strings.HasPrefix(data, "custom_time:"):
		slots.HandleCustomTime(ctx, b, callback, h)
	case strings.HasPrefix(data, "custom_period:"):
		slots.HandleCustomPeriod(ctx, b, callback, h)
	case strings.HasPrefix(data, "period_weeks:"):
		slots.HandlePeriodWeeks(ctx, b, callback, h)
	case strings.HasPrefix(data, "period_weekday:"):
		slots.HandlePeriodWeekday(ctx, b, callback, h)
	case strings.HasPrefix(data, "period_time:"):
		slots.HandlePeriodTime(ctx, b, callback, h)
	case strings.HasPrefix(data, "workday_day:"):
		slots.HandleWorkdayDay(ctx, b, callback, h)
	case strings.HasPrefix(data, "workday_start:"):
		slots.HandleWorkdayStart(ctx, b, callback, h)
	case strings.HasPrefix(data, "workday_end:"):
		slots.HandleWorkdayEnd(ctx, b, callback, h)
	case strings.HasPrefix(data, SetWeekday):
		slots.HandleSetWeekday(ctx, b, callback, h)
	case strings.HasPrefix(data, SetTime):
		slots.HandleSetTime(ctx, b, callback, h)
	case data == ManualBook:
		slots.HandleManualBook(ctx, b, callback, h)

	// ===== Student: Booking Lessons =====
	case strings.HasPrefix(data, ViewScheduleSubject):
		student.HandleViewScheduleSubject(ctx, b, callback, h)
	case strings.HasPrefix(data, "view_extended_slots:"):
		student.HandleViewExtendedSlots(ctx, b, callback, h)
	case strings.HasPrefix(data, "request_recurring_booking:"):
		student.HandleRequestRecurringBooking(ctx, b, callback, h)
	case strings.HasPrefix(data, "request_recurring_confirm:"):
		student.HandleRequestRecurringConfirm(ctx, b, callback, h)
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
	case strings.HasPrefix(data, "approve_recurring:"):
		recurring.HandleApproveRecurring(ctx, b, callback, h)
	case strings.HasPrefix(data, "reject_recurring:"):
		recurring.HandleRejectRecurring(ctx, b, callback, h)

	// ===== Student: Teacher Access Management =====
	case data == "subjects_menu":
		// Back to subjects menu - will be handled in the main command handler
		common.HandleBackToSubjects(ctx, b, callback, h)
	case data == "my_teachers":
		student.HandleMyTeachers(ctx, b, callback, h)
	case data == "public_teachers":
		student.HandlePublicTeachers(ctx, b, callback, h)
	case strings.HasPrefix(data, "public_teachers_page:"):
		page, err := common.ParseIDFromCallback(data)
		if err != nil {
			h.Logger.Error("Failed to parse page number", zap.Error(err))
			common.AnswerCallbackAlert(ctx, b, callback.ID, "❌ Неверный формат")
			return
		}
		student.HandlePublicTeachersPage(ctx, b, callback, h, int(page))
	case strings.HasPrefix(data, "teacher_profile:"):
		student.HandleTeacherProfile(ctx, b, callback, h)
	case data == "find_teacher":
		student.HandleFindTeacher(ctx, b, callback, h)
	case data == "enter_invite_code":
		student.HandleEnterInviteCode(ctx, b, callback, h)
	case data == "send_access_request":
		student.HandleSendAccessRequest(ctx, b, callback, h)
	case data == "my_requests":
		student.HandleMyRequests(ctx, b, callback, h)

	// ===== Teacher: Access Settings =====
	case data == "teacher_settings":
		teacher.HandleTeacherSettings(ctx, b, callback, h)
	case data == "toggle_public_status":
		teacher.HandleTogglePublicStatus(ctx, b, callback, h)
	case data == "manage_invite_codes":
		teacher.HandleManageInviteCodes(ctx, b, callback, h)
	case data == "create_invite_code":
		teacher.HandleCreateInviteCode(ctx, b, callback, h)
	case strings.HasPrefix(data, "deactivate_code:"):
		teacher.HandleDeactivateInviteCode(ctx, b, callback, h)
	case data == "view_access_requests":
		teacher.HandleViewAccessRequests(ctx, b, callback, h)
	case strings.HasPrefix(data, "approve_request:"):
		teacher.HandleApproveAccessRequest(ctx, b, callback, h)
	case strings.HasPrefix(data, "reject_request:"):
		teacher.HandleRejectAccessRequest(ctx, b, callback, h)
	case data == "view_my_students":
		teacher.HandleViewMyStudents(ctx, b, callback, h)
	case strings.HasPrefix(data, "revoke_access:"):
		teacher.HandleRevokeStudentAccess(ctx, b, callback, h)
	case data == "mysubjects":
		// Back to my subjects - will be handled in the main command handler
		if h.HandleMySubjects != nil {
			h.HandleMySubjects(ctx, b, &models.Update{CallbackQuery: callback})
		}

	// ===== Unknown Callback =====
	default:
		h.Logger.Warn("Unknown callback",
			zap.String("data", data),
			zap.Int64("user_id", callback.From.ID))
		common.AnswerCallback(ctx, b, callback.ID, "❌ Неизвестная команда")
	}

	h.Logger.Info("Callback routed successfully", zap.String("data", data))
}
