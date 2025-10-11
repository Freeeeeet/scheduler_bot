package service

import (
	"context"
	"fmt"

	"github.com/Freeeeeet/scheduler_bot/internal/model"
	"github.com/Freeeeeet/scheduler_bot/internal/repository"
	"go.uber.org/zap"
)

type UserService struct {
	userRepo *repository.UserRepository
	logger   *zap.Logger
}

func NewUserService(userRepo *repository.UserRepository, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// RegisterUser регистрирует или обновляет пользователя
func (s *UserService) RegisterUser(ctx context.Context, telegramID int64, username, firstName, lastName, languageCode string) (*model.User, error) {
	// Проверяем существует ли пользователь
	existingUser, err := s.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}

	// Если пользователь уже существует, обновляем данные
	if existingUser != nil {
		existingUser.Username = username
		existingUser.FirstName = firstName
		existingUser.LastName = lastName
		existingUser.LanguageCode = languageCode

		err = s.userRepo.Update(ctx, existingUser)
		if err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}

		s.logger.Info("User updated",
			zap.Int64("telegram_id", telegramID),
			zap.String("username", username),
		)

		return existingUser, nil
	}

	// Создаём нового пользователя
	user := &model.User{
		TelegramID:   telegramID,
		Username:     username,
		FirstName:    firstName,
		LastName:     lastName,
		LanguageCode: languageCode,
		IsTeacher:    false, // По умолчанию студент
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	s.logger.Info("New user registered",
		zap.Int64("user_id", user.ID),
		zap.Int64("telegram_id", telegramID),
		zap.String("username", username),
	)

	return user, nil
}

// GetByTelegramID получает пользователя по Telegram ID
func (s *UserService) GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	return s.userRepo.GetByTelegramID(ctx, telegramID)
}

// GetByID получает пользователя по ID
func (s *UserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// MakeTeacher делает пользователя учителем
func (s *UserService) MakeTeacher(ctx context.Context, telegramID int64) error {
	user, err := s.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	user.IsTeacher = true
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	s.logger.Info("User became teacher",
		zap.Int64("user_id", user.ID),
		zap.String("username", user.Username),
	)

	return nil
}
