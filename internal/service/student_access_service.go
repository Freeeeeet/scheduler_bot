package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/Freeeeeet/scheduler_bot/internal/repository"
	"go.uber.org/zap"
)

type StudentAccessService struct {
	accessRepo     *repository.AccessRepository
	inviteCodeRepo *repository.InviteCodeRepository
	requestRepo    *repository.AccessRequestRepository
	userRepo       *repository.UserRepository
	subjectRepo    *repository.SubjectRepository
	logger         *zap.Logger
}

func NewStudentAccessService(
	accessRepo *repository.AccessRepository,
	inviteCodeRepo *repository.InviteCodeRepository,
	requestRepo *repository.AccessRequestRepository,
	userRepo *repository.UserRepository,
	subjectRepo *repository.SubjectRepository,
	logger *zap.Logger,
) *StudentAccessService {
	return &StudentAccessService{
		accessRepo:     accessRepo,
		inviteCodeRepo: inviteCodeRepo,
		requestRepo:    requestRepo,
		userRepo:       userRepo,
		subjectRepo:    subjectRepo,
		logger:         logger,
	}
}

// ============ Проверки доступа ============

// CanStudentSeeTeacher проверяет, может ли студент видеть учителя
func (s *StudentAccessService) CanStudentSeeTeacher(ctx context.Context, studentID, teacherID int64) (bool, error) {
	// Получаем учителя
	teacher, err := s.userRepo.GetByID(ctx, teacherID)
	if err != nil {
		return false, fmt.Errorf("get teacher: %w", err)
	}

	if teacher == nil || !teacher.IsTeacher {
		return false, nil
	}

	// Если учитель публичный, доступ есть у всех
	if teacher.IsPublic {
		return true, nil
	}

	// Иначе проверяем наличие записи в student_teacher_access
	hasAccess, err := s.accessRepo.HasAccess(ctx, studentID, teacherID)
	if err != nil {
		return false, fmt.Errorf("check access: %w", err)
	}

	return hasAccess, nil
}

// GetMyTeachers получает список "Мои учителя" для студента
func (s *StudentAccessService) GetMyTeachers(ctx context.Context, studentID int64) ([]*model.User, error) {
	// Получаем ID учителей с доступом
	teacherIDs, err := s.accessRepo.GetStudentTeacherIDs(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student teacher ids: %w", err)
	}

	if len(teacherIDs) == 0 {
		return []*model.User{}, nil
	}

	// Получаем полную информацию об учителях
	teachers, err := s.userRepo.GetByIDs(ctx, teacherIDs)
	if err != nil {
		return nil, fmt.Errorf("get teachers: %w", err)
	}

	return teachers, nil
}

// GetPublicTeachers получает список публичных учителей
func (s *StudentAccessService) GetPublicTeachers(ctx context.Context) ([]*model.User, error) {
	teachers, err := s.userRepo.GetPublicTeachers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get public teachers: %w", err)
	}

	return teachers, nil
}

// GetAccessibleSubjects получает предметы доступных учителей (для студента)
func (s *StudentAccessService) GetAccessibleSubjects(ctx context.Context, studentID int64) ([]*model.Subject, error) {
	// Получаем публичные предметы
	publicSubjects, err := s.subjectRepo.GetPublicActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("get public subjects: %w", err)
	}

	// Получаем учителей с доступом
	teacherIDs, err := s.accessRepo.GetStudentTeacherIDs(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student teacher ids: %w", err)
	}

	// Получаем предметы учителей с доступом
	var privateSubjects []*model.Subject
	if len(teacherIDs) > 0 {
		privateSubjects, err = s.subjectRepo.GetActiveByTeacherIDs(ctx, teacherIDs)
		if err != nil {
			return nil, fmt.Errorf("get private subjects: %w", err)
		}
	}

	// Объединяем и удаляем дубликаты
	subjectMap := make(map[int64]*model.Subject)
	for _, subject := range publicSubjects {
		subjectMap[subject.ID] = subject
	}
	for _, subject := range privateSubjects {
		subjectMap[subject.ID] = subject
	}

	subjects := make([]*model.Subject, 0, len(subjectMap))
	for _, subject := range subjectMap {
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

// ============ Invite-коды ============

// generateInviteCode генерирует уникальный invite-код
func (s *StudentAccessService) generateInviteCode(ctx context.Context) (string, error) {
	const maxAttempts = 10

	for i := 0; i < maxAttempts; i++ {
		// Генерируем 8 случайных байт
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate random bytes: %w", err)
		}

		// Кодируем в base32 и берем первые 8 символов
		code := base32.StdEncoding.EncodeToString(bytes)
		code = strings.TrimRight(code, "=") // Убираем padding
		if len(code) > 8 {
			code = code[:8]
		}

		// Проверяем уникальность
		exists, err := s.inviteCodeRepo.CodeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("check code exists: %w", err)
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxAttempts)
}

// CreateInviteCode создает invite-код для учителя
func (s *StudentAccessService) CreateInviteCode(ctx context.Context, teacherID int64, maxUses *int, expiresAt *time.Time) (*model.TeacherInviteCode, error) {
	// Проверяем, что пользователь - учитель
	teacher, err := s.userRepo.GetByID(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get teacher: %w", err)
	}

	if teacher == nil || !teacher.IsTeacher {
		return nil, fmt.Errorf("user is not a teacher")
	}

	// Генерируем уникальный код
	code, err := s.generateInviteCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate invite code: %w", err)
	}

	// Создаем код
	inviteCode := &model.TeacherInviteCode{
		TeacherID: teacherID,
		Code:      code,
		MaxUses:   maxUses,
		ExpiresAt: expiresAt,
		IsActive:  true,
	}

	err = s.inviteCodeRepo.Create(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("create invite code: %w", err)
	}

	s.logger.Info("Invite code created",
		zap.Int64("teacher_id", teacherID),
		zap.String("code", code),
	)

	return inviteCode, nil
}

// UseInviteCode использует invite-код студентом
func (s *StudentAccessService) UseInviteCode(ctx context.Context, studentID int64, code string) error {
	// Получаем код
	inviteCode, err := s.inviteCodeRepo.GetByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("get invite code: %w", err)
	}

	if inviteCode == nil {
		return fmt.Errorf("invite code not found")
	}

	// Проверяем валидность
	if !inviteCode.IsValid() {
		return fmt.Errorf("invite code is not valid")
	}

	// Проверяем, нет ли уже доступа
	hasAccess, err := s.accessRepo.HasAccess(ctx, studentID, inviteCode.TeacherID)
	if err != nil {
		return fmt.Errorf("check access: %w", err)
	}

	if hasAccess {
		return fmt.Errorf("access already granted")
	}

	// Предоставляем доступ
	err = s.accessRepo.GrantAccess(ctx, studentID, inviteCode.TeacherID, model.AccessTypeInvited)
	if err != nil {
		return fmt.Errorf("grant access: %w", err)
	}

	// Инкрементируем использования
	err = s.inviteCodeRepo.UseCode(ctx, inviteCode.ID)
	if err != nil {
		s.logger.Error("Failed to increment code usage",
			zap.Int64("code_id", inviteCode.ID),
			zap.Error(err),
		)
		// Не возвращаем ошибку, т.к. доступ уже предоставлен
	}

	s.logger.Info("Invite code used",
		zap.Int64("student_id", studentID),
		zap.Int64("teacher_id", inviteCode.TeacherID),
		zap.String("code", code),
	)

	return nil
}

// GetTeacherInviteCodes получает коды учителя
func (s *StudentAccessService) GetTeacherInviteCodes(ctx context.Context, teacherID int64) ([]*model.TeacherInviteCode, error) {
	codes, err := s.inviteCodeRepo.GetByTeacherID(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get teacher invite codes: %w", err)
	}

	return codes, nil
}

// DeactivateInviteCode деактивирует код
func (s *StudentAccessService) DeactivateInviteCode(ctx context.Context, teacherID, codeID int64) error {
	// Проверяем, что код принадлежит учителю
	code, err := s.inviteCodeRepo.GetByID(ctx, codeID)
	if err != nil {
		return fmt.Errorf("get invite code: %w", err)
	}

	if code == nil {
		return fmt.Errorf("invite code not found")
	}

	if code.TeacherID != teacherID {
		return fmt.Errorf("access denied: code belongs to another teacher")
	}

	// Деактивируем
	err = s.inviteCodeRepo.Deactivate(ctx, codeID)
	if err != nil {
		return fmt.Errorf("deactivate code: %w", err)
	}

	s.logger.Info("Invite code deactivated",
		zap.Int64("teacher_id", teacherID),
		zap.Int64("code_id", codeID),
	)

	return nil
}

// ============ Заявки на доступ ============

// CreateAccessRequest создает заявку на доступ
func (s *StudentAccessService) CreateAccessRequest(ctx context.Context, studentID, teacherID int64, message string) error {
	// Проверяем, что учитель существует
	teacher, err := s.userRepo.GetByID(ctx, teacherID)
	if err != nil {
		return fmt.Errorf("get teacher: %w", err)
	}

	if teacher == nil || !teacher.IsTeacher {
		return fmt.Errorf("teacher not found")
	}

	// Проверяем, нет ли уже доступа
	hasAccess, err := s.accessRepo.HasAccess(ctx, studentID, teacherID)
	if err != nil {
		return fmt.Errorf("check access: %w", err)
	}

	if hasAccess {
		return fmt.Errorf("access already granted")
	}

	// Проверяем, нет ли pending заявки
	hasPending, err := s.requestRepo.HasPendingRequest(ctx, studentID, teacherID)
	if err != nil {
		return fmt.Errorf("check pending request: %w", err)
	}

	if hasPending {
		return fmt.Errorf("pending request already exists")
	}

	// Создаем заявку
	request := &model.AccessRequest{
		StudentID: studentID,
		TeacherID: teacherID,
		Status:    model.RequestStatusPending,
		Message:   message,
	}

	err = s.requestRepo.Create(ctx, request)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	s.logger.Info("Access request created",
		zap.Int64("student_id", studentID),
		zap.Int64("teacher_id", teacherID),
		zap.Int64("request_id", request.ID),
	)

	return nil
}

// ApproveAccessRequest одобряет заявку (учитель)
func (s *StudentAccessService) ApproveAccessRequest(ctx context.Context, teacherID, requestID int64, response string) error {
	// Получаем заявку
	request, err := s.requestRepo.GetByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("get request: %w", err)
	}

	if request == nil {
		return fmt.Errorf("request not found")
	}

	// Проверяем, что заявка к этому учителю
	if request.TeacherID != teacherID {
		return fmt.Errorf("access denied: request belongs to another teacher")
	}

	// Проверяем, что заявка pending
	if !request.IsPending() {
		return fmt.Errorf("request is not pending")
	}

	// Обновляем статус
	err = s.requestRepo.UpdateStatus(ctx, requestID, model.RequestStatusApproved, response)
	if err != nil {
		return fmt.Errorf("update request status: %w", err)
	}

	// Предоставляем доступ
	err = s.accessRepo.GrantAccess(ctx, request.StudentID, teacherID, model.AccessTypeApproved)
	if err != nil {
		return fmt.Errorf("grant access: %w", err)
	}

	s.logger.Info("Access request approved",
		zap.Int64("request_id", requestID),
		zap.Int64("student_id", request.StudentID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}

// RejectAccessRequest отклоняет заявку (учитель)
func (s *StudentAccessService) RejectAccessRequest(ctx context.Context, teacherID, requestID int64, response string) error {
	// Получаем заявку
	request, err := s.requestRepo.GetByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("get request: %w", err)
	}

	if request == nil {
		return fmt.Errorf("request not found")
	}

	// Проверяем, что заявка к этому учителю
	if request.TeacherID != teacherID {
		return fmt.Errorf("access denied: request belongs to another teacher")
	}

	// Проверяем, что заявка pending
	if !request.IsPending() {
		return fmt.Errorf("request is not pending")
	}

	// Обновляем статус
	err = s.requestRepo.UpdateStatus(ctx, requestID, model.RequestStatusRejected, response)
	if err != nil {
		return fmt.Errorf("update request status: %w", err)
	}

	s.logger.Info("Access request rejected",
		zap.Int64("request_id", requestID),
		zap.Int64("student_id", request.StudentID),
		zap.Int64("teacher_id", teacherID),
	)

	return nil
}

// GetPendingRequests получает pending заявки учителя
func (s *StudentAccessService) GetPendingRequests(ctx context.Context, teacherID int64) ([]*model.AccessRequest, error) {
	requests, err := s.requestRepo.GetPendingByTeacher(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get pending requests: %w", err)
	}

	return requests, nil
}

// GetStudentRequests получает заявки студента
func (s *StudentAccessService) GetStudentRequests(ctx context.Context, studentID int64) ([]*model.AccessRequest, error) {
	requests, err := s.requestRepo.GetByStudent(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student requests: %w", err)
	}

	return requests, nil
}

// ============ Управление доступом ============

// RevokeStudentAccess отзывает доступ у студента (учитель)
func (s *StudentAccessService) RevokeStudentAccess(ctx context.Context, teacherID, studentID int64) error {
	// Проверяем, что доступ существует
	hasAccess, err := s.accessRepo.HasAccess(ctx, studentID, teacherID)
	if err != nil {
		return fmt.Errorf("check access: %w", err)
	}

	if !hasAccess {
		return fmt.Errorf("access not found")
	}

	// Отзываем доступ
	err = s.accessRepo.RevokeAccess(ctx, studentID, teacherID)
	if err != nil {
		return fmt.Errorf("revoke access: %w", err)
	}

	s.logger.Info("Access revoked",
		zap.Int64("teacher_id", teacherID),
		zap.Int64("student_id", studentID),
	)

	return nil
}

// GetMyStudents получает список студентов учителя
func (s *StudentAccessService) GetMyStudents(ctx context.Context, teacherID int64) ([]*model.User, error) {
	// Получаем ID студентов
	studentIDs, err := s.accessRepo.GetTeacherStudentIDs(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("get teacher student ids: %w", err)
	}

	if len(studentIDs) == 0 {
		return []*model.User{}, nil
	}

	// Получаем полную информацию о студентах
	students, err := s.userRepo.GetByIDs(ctx, studentIDs)
	if err != nil {
		return nil, fmt.Errorf("get students: %w", err)
	}

	return students, nil
}

// CountPendingRequests подсчитывает pending заявки учителя
func (s *StudentAccessService) CountPendingRequests(ctx context.Context, teacherID int64) (int, error) {
	count, err := s.requestRepo.CountPendingByTeacher(ctx, teacherID)
	if err != nil {
		return 0, fmt.Errorf("count pending requests: %w", err)
	}

	return count, nil
}

// CountStudents подсчитывает студентов учителя
func (s *StudentAccessService) CountStudents(ctx context.Context, teacherID int64) (int, error) {
	count, err := s.accessRepo.CountTeacherStudents(ctx, teacherID)
	if err != nil {
		return 0, fmt.Errorf("count students: %w", err)
	}

	return count, nil
}
