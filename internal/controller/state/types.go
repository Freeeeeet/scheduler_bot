package state

// UserState представляет текущее состояние пользователя в диалоге
type UserState string

const (
	StateNone UserState = "" // Нет активного состояния

	// Состояния для создания предмета
	StateCreateSubjectName        UserState = "create_subject_name"
	StateCreateSubjectDescription UserState = "create_subject_description"
	StateCreateSubjectPrice       UserState = "create_subject_price"
	StateCreateSubjectDuration    UserState = "create_subject_duration"
	StateCreateSubjectApproval    UserState = "create_subject_approval"

	// Состояния для редактирования предмета
	StateEditSubjectName        UserState = "edit_subject_name"
	StateEditSubjectDescription UserState = "edit_subject_description"
	StateEditSubjectPrice       UserState = "edit_subject_price"
	StateEditSubjectDuration    UserState = "edit_subject_duration"

	// Состояния для добавления слотов
	StateAddSlotsSubjectID UserState = "add_slots_subject_id"
	StateAddSlotsWeekday   UserState = "add_slots_weekday"
	StateAddSlotsTime      UserState = "add_slots_time"
	StateAddSlotsDuration  UserState = "add_slots_duration"

	// Состояния для студентов - доступ
	StateEnteringInviteCode    UserState = "entering_invite_code"
	StateEnteringAccessMessage UserState = "entering_access_message"
	StateSearchingTeacher      UserState = "searching_teacher"

	// Состояния для учителей - управление доступом
	StateCreatingInviteCode  UserState = "creating_invite_code"
	StateRespondingToRequest UserState = "responding_to_request"

	// Состояния для пометки слотов занятыми
	StateMarkSlotBusyComment UserState = "mark_slot_busy_comment"
)

// UserData хранит временные данные пользователя во время диалога
type UserData struct {
	State UserState
	Data  map[string]interface{} // Временные данные для текущего диалога
}
