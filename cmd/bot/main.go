package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/Freeeeeet/scheduler_bot/internal/app"
	"github.com/Freeeeeet/scheduler_bot/internal/config"
	"github.com/Freeeeeet/scheduler_bot/internal/controller"
	"github.com/Freeeeeet/scheduler_bot/internal/repository"
	"github.com/Freeeeeet/scheduler_bot/internal/service"
	"github.com/go-telegram/bot"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализация логгера
	logger := app.NewLogger(cfg.Environment)
	defer logger.Sync()

	logger.Info("Starting Scheduler Bot", zap.String("env", cfg.Environment))

	// Создаём контекст для приложения
	ctx := context.Background()

	// Подключение к базе данных через пул соединений
	pool, err := pgxpool.New(ctx, cfg.GetDBDSN())
	if err != nil {
		logger.Fatal("Failed to create connection pool", zap.Error(err))
	}
	defer pool.Close()

	// Проверка соединения с БД
	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("Database connection failed", zap.Error(err))
	}

	// Получаем статистику пула
	stat := pool.Stat()
	logger.Info("✅ Database connection pool established",
		zap.Int32("total_conns", stat.TotalConns()),
		zap.Int32("idle_conns", stat.IdleConns()),
		zap.Int32("max_conns", stat.MaxConns()),
	)

	// Применение миграций
	migrationsPath := getMigrationsPath()
	migrator, err := app.NewMigrator(pool, migrationsPath)
	if err != nil {
		logger.Fatal("Failed to create migrator", zap.Error(err))
	}
	defer migrator.Close()

	if err := migrator.Run(ctx); err != nil {
		logger.Fatal("Migration failed", zap.Error(err))
	}

	// Показываем текущую версию миграций
	version, err := migrator.Version(ctx)
	if err != nil {
		logger.Fatal("Failed to get migration version", zap.Error(err))
	}

	logger.Info("📊 Database version", zap.Int64("version", version))

	// Инициализация репозиториев
	userRepo := repository.NewUserRepository(pool)
	subjectRepo := repository.NewSubjectRepository(pool, logger)
	slotRepo := repository.NewSlotRepository(pool)
	bookingRepo := repository.NewBookingRepository(pool)
	recurringRepo := repository.NewRecurringScheduleRepository(pool, logger)
	accessRepo := repository.NewAccessRepository(pool)
	inviteCodeRepo := repository.NewInviteCodeRepository(pool)
	accessRequestRepo := repository.NewAccessRequestRepository(pool)

	logger.Info("✅ Repositories initialized")

	// Инициализация сервисов
	userService := service.NewUserService(userRepo, logger)
	bookingService := service.NewBookingService(pool, userRepo, subjectRepo, slotRepo, bookingRepo, logger)
	teacherService := service.NewTeacherService(userRepo, subjectRepo, slotRepo, bookingRepo, recurringRepo, logger)
	accessService := service.NewStudentAccessService(accessRepo, inviteCodeRepo, accessRequestRepo, userRepo, subjectRepo, logger)

	logger.Info("✅ Services initialized")

	// Создание Telegram бота
	botInstance, err := bot.New(cfg.TelegramToken)
	if err != nil {
		logger.Fatal("❌ Failed to create bot", zap.Error(err))
	}

	logger.Info("✅ Telegram bot created")

	// Инициализация контроллера
	botController := controller.NewBotController(
		botInstance,
		userService,
		bookingService,
		teacherService,
		accessService,
		userRepo,
		inviteCodeRepo,
		accessRepo,
		accessRequestRepo,
		logger,
	)

	// Регистрируем handlers и устанавливаем меню команд
	if err := botController.RegisterHandlers(ctx); err != nil {
		logger.Fatal("❌ Failed to register handlers", zap.Error(err))
	}

	logger.Info("✅ Bot handlers registered")

	// Запуск фонового планировщика для автоматической генерации слотов
	scheduler := app.NewScheduler(teacherService, logger)
	scheduler.Start(ctx)
	logger.Info("✅ Background scheduler started")

	logger.Info("🚀 Bot is starting...")

	// Запуск бота
	if err := botController.Start(ctx); err != nil {
		logger.Fatal("❌ Bot failed to start", zap.Error(err))
	}
}

// getMigrationsPath возвращает путь к папке с миграциями
func getMigrationsPath() string {
	// Пытаемся найти папку migrations относительно текущей директории
	possiblePaths := []string{
		"./migrations",
		"./../migrations", // если запускаем из cmd/bot
		"./../../migrations",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}

	// Если не нашли, используем текущую директорию
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "migrations")
}
