# ТЕХНИЧЕСКОЕ ЗАДАНИЕ: Система видимости преподавателей и улучшенный интерфейс студентов

## 1. ЦЕЛЬ
Реализовать гибридную систему доступа студентов к преподавателям с разделением на публичных/приватных учителей, системой invite-кодов и заявок на доступ.

---

## 2. АРХИТЕКТУРА РЕШЕНИЯ

### 2.1. Изменения в базе данных

#### Миграция 1: Добавление публичности учителя
```sql
-- Таблица users
ALTER TABLE users ADD COLUMN is_public BOOLEAN DEFAULT FALSE;
COMMENT ON COLUMN users.is_public IS 'Публичный учитель (виден всем) или приватный (нужен доступ)';
CREATE INDEX idx_users_is_public ON users(is_public) WHERE is_public = true AND is_teacher = true;
```

#### Миграция 2: Таблица связей студент-учитель
```sql
CREATE TABLE student_teacher_access (
    id BIGSERIAL PRIMARY KEY,
    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_type TEXT NOT NULL, -- 'invited', 'approved', 'subscribed'
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_student_teacher UNIQUE (student_id, teacher_id),
    CONSTRAINT valid_access_type CHECK (access_type IN ('invited', 'approved', 'subscribed'))
);

CREATE INDEX idx_access_student ON student_teacher_access(student_id);
CREATE INDEX idx_access_teacher ON student_teacher_access(teacher_id);

COMMENT ON TABLE student_teacher_access IS 'Доступ студентов к приватным учителям';
COMMENT ON COLUMN student_teacher_access.access_type IS 'invited=по коду, approved=одобрена заявка, subscribed=подписка';
```

#### Миграция 3: Таблица invite-кодов
```sql
CREATE TABLE teacher_invite_codes (
    id BIGSERIAL PRIMARY KEY,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL UNIQUE,
    max_uses INTEGER DEFAULT NULL, -- NULL = безлимит
    current_uses INTEGER DEFAULT 0,
    expires_at TIMESTAMPTZ DEFAULT NULL, -- NULL = не истекает
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_max_uses CHECK (max_uses IS NULL OR max_uses > 0),
    CONSTRAINT valid_current_uses CHECK (current_uses >= 0)
);

CREATE INDEX idx_invite_codes_teacher ON teacher_invite_codes(teacher_id);
CREATE INDEX idx_invite_codes_code ON teacher_invite_codes(code) WHERE is_active = true;

COMMENT ON TABLE teacher_invite_codes IS 'Пригласительные коды для доступа к приватным учителям';
```

#### Миграция 4: Таблица заявок на доступ
```sql
CREATE TABLE access_requests (
    id BIGSERIAL PRIMARY KEY,
    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    teacher_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    message TEXT, -- Сообщение от студента
    teacher_response TEXT, -- Ответ учителя
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    
    CONSTRAINT valid_status CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT unique_pending_request UNIQUE (student_id, teacher_id, status)
);

CREATE INDEX idx_requests_student ON access_requests(student_id, status);
CREATE INDEX idx_requests_teacher ON access_requests(teacher_id, status);
CREATE INDEX idx_requests_pending ON access_requests(teacher_id) WHERE status = 'pending';

COMMENT ON TABLE access_requests IS 'Заявки студентов на доступ к приватным учителям';
```

---

## 3. BACKEND: МОДЕЛИ И РЕПОЗИТОРИИ

### 3.1. Модель User (изменения)
```go
// internal/model/user.go
type User struct {
    ID                  int64     `json:"id"`
    TelegramID          int64     `json:"telegram_id"`
    Username            string    `json:"username"`
    FirstName           string    `json:"first_name"`
    LastName            string    `json:"last_name"`
    LanguageCode        string    `json:"language_code"`
    IsTeacher           bool      `json:"is_teacher"`
    IsPublic            bool      `json:"is_public"` // НОВОЕ
    AutoApproveBookings bool      `json:"auto_approve_bookings"`
    CreatedAt           time.Time `json:"created_at"`
}
```

### 3.2. Новые модели
```go
// internal/model/student_teacher_access.go
type StudentTeacherAccess struct {
    ID         int64     `json:"id"`
    StudentID  int64     `json:"student_id"`
    TeacherID  int64     `json:"teacher_id"`
    AccessType string    `json:"access_type"` // 'invited', 'approved', 'subscribed'
    GrantedAt  time.Time `json:"granted_at"`
}

// internal/model/teacher_invite_code.go
type TeacherInviteCode struct {
    ID          int64      `json:"id"`
    TeacherID   int64      `json:"teacher_id"`
    Code        string     `json:"code"`
    MaxUses     *int       `json:"max_uses"`      // nil = безлимит
    CurrentUses int        `json:"current_uses"`
    ExpiresAt   *time.Time `json:"expires_at"`    // nil = не истекает
    IsActive    bool       `json:"is_active"`
    CreatedAt   time.Time  `json:"created_at"`
}

// internal/model/access_request.go
type AccessRequest struct {
    ID              int64      `json:"id"`
    StudentID       int64      `json:"student_id"`
    TeacherID       int64      `json:"teacher_id"`
    Status          string     `json:"status"` // 'pending', 'approved', 'rejected'
    Message         string     `json:"message"`
    TeacherResponse string     `json:"teacher_response"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       *time.Time `json:"updated_at"`
}
```

### 3.3. Новые репозитории

#### AccessRepository
```go
// internal/repository/access_repository.go
type AccessRepository struct {
    *base.Repository
}

func NewAccessRepository(db *sql.DB, logger *zap.Logger) *AccessRepository

// Проверяет, есть ли у студента доступ к учителю
func (r *AccessRepository) HasAccess(ctx context.Context, studentID, teacherID int64) (bool, error)

// Предоставляет доступ
func (r *AccessRepository) GrantAccess(ctx context.Context, studentID, teacherID int64, accessType string) error

// Отзывает доступ
func (r *AccessRepository) RevokeAccess(ctx context.Context, studentID, teacherID int64) error

// Получает всех учителей студента
func (r *AccessRepository) GetStudentTeachers(ctx context.Context, studentID int64) ([]*model.User, error)

// Получает всех студентов учителя
func (r *AccessRepository) GetTeacherStudents(ctx context.Context, teacherID int64) ([]*model.User, error)
```

#### InviteCodeRepository
```go
// internal/repository/invite_code_repository.go
type InviteCodeRepository struct {
    *base.Repository
}

func NewInviteCodeRepository(db *sql.DB, logger *zap.Logger) *InviteCodeRepository

// Создает новый invite-код
func (r *InviteCodeRepository) Create(ctx context.Context, code *model.TeacherInviteCode) error

// Получает код по строке
func (r *InviteCodeRepository) GetByCode(ctx context.Context, code string) (*model.TeacherInviteCode, error)

// Получает все коды учителя
func (r *InviteCodeRepository) GetByTeacherID(ctx context.Context, teacherID int64) ([]*model.TeacherInviteCode, error)

// Использует код (инкремент current_uses)
func (r *InviteCodeRepository) UseCode(ctx context.Context, codeID int64) error

// Деактивирует код
func (r *InviteCodeRepository) Deactivate(ctx context.Context, codeID int64) error

// Удаляет код
func (r *InviteCodeRepository) Delete(ctx context.Context, codeID int64) error

// Проверяет валидность кода
func (r *InviteCodeRepository) IsValid(ctx context.Context, code string) (bool, error)
```

#### AccessRequestRepository
```go
// internal/repository/access_request_repository.go
type AccessRequestRepository struct {
    *base.Repository
}

func NewAccessRequestRepository(db *sql.DB, logger *zap.Logger) *AccessRequestRepository

// Создает заявку
func (r *AccessRequestRepository) Create(ctx context.Context, req *model.AccessRequest) error

// Получает заявку по ID
func (r *AccessRequestRepository) GetByID(ctx context.Context, id int64) (*model.AccessRequest, error)

// Получает pending заявки учителя
func (r *AccessRequestRepository) GetPendingByTeacher(ctx context.Context, teacherID int64) ([]*model.AccessRequest, error)

// Получает заявки студента
func (r *AccessRequestRepository) GetByStudent(ctx context.Context, studentID int64) ([]*model.AccessRequest, error)

// Проверяет, есть ли активная заявка
func (r *AccessRequestRepository) HasPendingRequest(ctx context.Context, studentID, teacherID int64) (bool, error)

// Обновляет статус заявки
func (r *AccessRequestRepository) UpdateStatus(ctx context.Context, id int64, status, response string) error
```

---

## 4. BACKEND: СЕРВИСНЫЙ СЛОЙ

### 4.1. Новый сервис: StudentAccessService
```go
// internal/service/student_access_service.go
type StudentAccessService struct {
    accessRepo      *repository.AccessRepository
    inviteCodeRepo  *repository.InviteCodeRepository
    requestRepo     *repository.AccessRequestRepository
    userRepo        *repository.UserRepository
    logger          *zap.Logger
}

func NewStudentAccessService(...) *StudentAccessService

// ============ Проверки доступа ============

// Проверяет, может ли студент видеть учителя
func (s *StudentAccessService) CanStudentSeeTeacher(ctx context.Context, studentID, teacherID int64) (bool, error)
// Логика: учитель публичный ИЛИ есть запись в student_teacher_access

// Получает список "Мои учителя" для студента
func (s *StudentAccessService) GetMyTeachers(ctx context.Context, studentID int64) ([]*model.User, error)

// Получает список публичных учителей
func (s *StudentAccessService) GetPublicTeachers(ctx context.Context) ([]*model.User, error)

// Получает предметы доступных учителей (для студента)
func (s *StudentAccessService) GetAccessibleSubjects(ctx context.Context, studentID int64) ([]*model.Subject, error)

// ============ Invite-коды ============

// Создает invite-код для учителя
func (s *StudentAccessService) CreateInviteCode(ctx context.Context, teacherID int64, maxUses *int, expiresAt *time.Time) (*model.TeacherInviteCode, error)
// Генерирует уникальный код (8 символов, base32)

// Использует invite-код студентом
func (s *StudentAccessService) UseInviteCode(ctx context.Context, studentID int64, code string) error
// Проверяет валидность → предоставляет доступ → инкремент uses

// Получает коды учителя
func (s *StudentAccessService) GetTeacherInviteCodes(ctx context.Context, teacherID int64) ([]*model.TeacherInviteCode, error)

// Деактивирует код
func (s *StudentAccessService) DeactivateInviteCode(ctx context.Context, teacherID, codeID int64) error

// ============ Заявки на доступ ============

// Создает заявку на доступ
func (s *StudentAccessService) CreateAccessRequest(ctx context.Context, studentID, teacherID int64, message string) error
// Проверяет: нет pending заявки, нет доступа, учитель существует

// Одобряет заявку (учитель)
func (s *StudentAccessService) ApproveAccessRequest(ctx context.Context, teacherID, requestID int64, response string) error
// Обновляет статус → создает запись в student_teacher_access → уведомляет студента

// Отклоняет заявку (учитель)
func (s *StudentAccessService) RejectAccessRequest(ctx context.Context, teacherID, requestID int64, response string) error

// Получает pending заявки учителя
func (s *StudentAccessService) GetPendingRequests(ctx context.Context, teacherID int64) ([]*model.AccessRequest, error)

// ============ Управление доступом ============

// Отзывает доступ у студента (учитель)
func (s *StudentAccessService) RevokeStudentAccess(ctx context.Context, teacherID, studentID int64) error

// Получает список студентов учителя
func (s *StudentAccessService) GetMyStudents(ctx context.Context, teacherID int64) ([]*model.User, error)
```

### 4.2. Изменения в TeacherService
```go
// internal/service/teacher_service.go

// УДАЛИТЬ или изменить:
// func (s *TeacherService) GetAllActiveSubjects(ctx context.Context) ([]*model.Subject, error)

// ДОБАВИТЬ:
// Получает предметы публичных учителей
func (s *TeacherService) GetPublicSubjects(ctx context.Context) ([]*model.Subject, error)

// Получает предметы конкретных учителей (по ID)
func (s *TeacherService) GetSubjectsByTeachers(ctx context.Context, teacherIDs []int64) ([]*model.Subject, error)
```

### 4.3. Изменения в SubjectRepository
```go
// internal/repository/subject_repository.go

// Получает активные предметы публичных учителей
func (r *SubjectRepository) GetPublicActive(ctx context.Context) ([]*model.Subject, error)

// Получает активные предметы списка учителей
func (r *SubjectRepository) GetActiveByTeacherIDs(ctx context.Context, teacherIDs []int64) ([]*model.Subject, error)
```

---

## 5. FRONTEND: ИНТЕРФЕЙС СТУДЕНТА

### 5.1. Команда /subjects (главное меню)
```
📚 Предметы и учителя

Выберите категорию:

[🎓 Мои учителя] - учителя, к которым у вас есть доступ
[🌍 Публичные учителя] - доступны всем студентам  
[🔍 Найти учителя] - по коду приглашения или имени
[📋 Мои заявки] - статус ваших запросов на доступ
```

### 5.2. "Мои учителя"
```
🎓 Мои учителя (3)

Учителя, к которым у вас есть доступ:

[👤 Иван Иванов] - Математика, Физика
[👤 Мария Петрова] - Английский язык
[👤 Сергей Сидоров] - Программирование

[⬅️ Назад]
```

При клике на учителя → список его предметов → выбор предмета → расписание

### 5.3. "Публичные учителя"
```
🌍 Публичные учителя

Показано: 1-5 из 23

[👤 Александр К.] - Математика (500₽/час)
[👤 Елена М.] - История (400₽/час)
[👤 Дмитрий П.] - Физика (600₽/час)

[◀️ Пред] [2] [3] [4] [5] [След ▶️]

💡 Фильтры:
[🔍 По предмету] [💰 По цене] [⭐ По рейтингу]

[⬅️ Назад в меню]
```

### 5.4. "Найти учителя"
```
🔍 Найти учителя

Варианты поиска:

[🎟️ У меня есть код приглашения]
[📝 Отправить заявку учителю]
[🔎 Поиск по имени/предмету]

[⬅️ Назад]
```

#### 5.4.1. Ввод кода приглашения
```
🎟️ Код приглашения

Введите код приглашения от учителя:

Пример: ABC12XYZ

[❌ Отмена]
```

После ввода кода:
- Валидация кода
- Проверка срока действия / лимита использований
- Предоставление доступа
- Уведомление: "✅ Доступ получен! Учитель [Имя] добавлен в 'Мои учителя'"

#### 5.4.2. Отправка заявки
```
📝 Отправить заявку учителю

Введите Telegram username или имя учителя:

[❌ Отмена]
```

После ввода → поиск учителей → выбор из списка → форма заявки:

```
📨 Заявка на доступ

Учитель: Иван Иванов
Предметы: Математика, Физика

Напишите сообщение учителю (необязательно):
(почему хотите учиться, опыт, цели и т.д.)

[✅ Отправить заявку] [❌ Отмена]
```

### 5.5. "Мои заявки"
```
📋 Мои заявки на доступ

⏳ Ожидают ответа (2):
- Иван Иванов (Математика) - отправлено 2 дня назад
- Мария Петрова (Английский) - отправлено 5 часов назад

✅ Одобрены (1):
- Сергей Сидоров - одобрено вчера
  Ответ: "Добро пожаловать!"

❌ Отклонены (0):

[⬅️ Назад]
```

---

## 6. FRONTEND: ИНТЕРФЕЙС УЧИТЕЛЯ

### 6.1. Настройки учителя (новый раздел в /mysubjects или отдельная команда)
```
⚙️ Настройки учителя

Видимость профиля:
[✅ Публичный] [⬜ Приватный]

Публичный: любой студент может найти вас и записаться
Приватный: доступ только по приглашению или заявке

───────────────

📊 Мои студенты: 12
📩 Заявки на доступ: 3 новых
🎟️ Коды приглашения: управление

[📩 Смотреть заявки]
[🎟️ Управление кодами]
[👥 Мои студенты]

[⬅️ Назад]
```

### 6.2. Управление кодами приглашения
```
🎟️ Коды приглашения

Активные коды (2):

1. ABC12XYZ
   Использований: 3/10
   Создан: 5 дней назад
   [📋 Копировать] [❌ Деактивировать]

2. XYZ98ABC
   Использований: 7/∞
   Истекает: через 20 дней
   [📋 Копировать] [❌ Деактивировать]

[➕ Создать новый код]

Неактивные коды (1):
[📂 Показать]

[⬅️ Назад]
```

#### Создание нового кода:
```
➕ Создать код приглашения

Настройки кода:

Лимит использований:
○ Без ограничений
○ Ограничить количеством: [___] человек

Срок действия:
○ Бессрочный
○ До даты: [выбрать дату]

[✅ Создать код] [❌ Отмена]
```

### 6.3. Заявки на доступ
```
📩 Заявки на доступ (3)

⏳ Новая заявка от Петр Иванов (@petrov)
Отправлена: 2 часа назад

Сообщение:
"Здравствуйте! Хочу изучать математику для подготовки к ЕГЭ. 
Занимаюсь самостоятельно уже полгода."

[✅ Одобрить] [❌ Отклонить]

───────────────

⏳ Новая заявка от Анна Смирнова (@anna_s)
Отправлена: 1 день назад

Сообщение не указано

[✅ Одобрить] [❌ Отклонить]

───────────────

[⬅️ Назад]
```

При одобрении:
```
✅ Одобрить заявку

Студент: Петр Иванов

Напишите приветственное сообщение (необязательно):

[Отправить одобрение] [Отмена]
```

### 6.4. Список студентов
```
👥 Мои студенты (12)

Поиск: [________] 🔍

1. Петр Иванов (@petrov)
   Доступ: по коду ABC12XYZ
   Записей: 5
   [📊 Статистика] [❌ Отозвать доступ]

2. Анна Смирнова (@anna_s)
   Доступ: одобрена заявка
   Записей: 12
   [📊 Статистика] [❌ Отозвать доступ]

...

[1] [2] [3] - страницы

[⬅️ Назад]
```

---

## 7. CALLBACK HANDLERS

### 7.1. Новые константы в router.go
```go
// Student access management
const (
    ViewMyTeachers      = "view_my_teachers"
    ViewPublicTeachers  = "view_public_teachers"
    ViewPublicTeachersPage = "view_public_teachers_page:" // page number
    FindTeacher         = "find_teacher"
    EnterInviteCode     = "enter_invite_code"
    SendAccessRequest   = "send_access_request"
    ViewMyRequests      = "view_my_requests"
    ViewTeacherProfile  = "view_teacher_profile:" // teacher_id
    
    // Teacher access management
    TogglePublicStatus     = "toggle_public_status"
    ManageInviteCodes      = "manage_invite_codes"
    CreateInviteCode       = "create_invite_code"
    CopyInviteCode         = "copy_invite_code:" // code_id
    DeactivateInviteCode   = "deactivate_invite_code:" // code_id
    ViewAccessRequests     = "view_access_requests"
    ApproveAccessRequest   = "approve_access_request:" // request_id
    RejectAccessRequest    = "reject_access_request:" // request_id
    ViewMyStudents         = "view_my_students"
    ViewMyStudentsPage     = "view_my_students_page:" // page
    RevokeStudentAccess    = "revoke_student_access:" // student_id
)
```

### 7.2. Новые handlers (структура файлов)
```
internal/controller/callbacks/student/
  - access.go          // ViewMyTeachers, FindTeacher, EnterInviteCode
  - teachers.go        // ViewPublicTeachers, ViewTeacherProfile
  - requests.go        // SendAccessRequest, ViewMyRequests

internal/controller/callbacks/teacher/
  - access_settings.go // TogglePublicStatus, ManageInviteCodes
  - students.go        // ViewAccessRequests, ViewMyStudents, RevokeStudentAccess
```

---

## 8. STATE MANAGEMENT

### 8.1. Новые состояния
```go
// internal/controller/state/types.go
const (
    // ... существующие состояния
    
    StateEnteringInviteCode    State = "entering_invite_code"
    StateEnteringAccessMessage State = "entering_access_message"
    StateSearchingTeacher      State = "searching_teacher"
    StateCreatingInviteCode    State = "creating_invite_code"
    StateRespondingToRequest   State = "responding_to_request"
)
```

---

## 9. УВЕДОМЛЕНИЯ

### 9.1. Уведомления студенту
- ✅ Код приглашения использован успешно
- ✅ Заявка отправлена учителю
- ✅ Заявка одобрена (с сообщением от учителя)
- ❌ Заявка отклонена (с сообщением от учителя)
- ⚠️ Доступ отозван учителем

### 9.2. Уведомления учителю
- 📩 Новая заявка на доступ
- 🎟️ Новый студент использовал invite-код
- 📊 Еженедельная сводка (кол-во новых студентов)

---

## 10. ПОРЯДОК РЕАЛИЗАЦИИ

### Этап 1: База данных и модели (1-2 дня)
1. Создать 4 миграции
2. Обновить модель User
3. Создать новые модели (StudentTeacherAccess, TeacherInviteCode, AccessRequest)
4. Прогнать миграции на тест-базе

### Этап 2: Репозитории (2-3 дня)
1. AccessRepository
2. InviteCodeRepository
3. AccessRequestRepository
4. Обновить SubjectRepository (GetPublicActive, GetActiveByTeacherIDs)
5. Написать unit-тесты для репозиториев

### Этап 3: Сервисный слой (3-4 дня)
1. Создать StudentAccessService
2. Обновить TeacherService
3. Интеграционные тесты для сервисов

### Этап 4: Интерфейс студента (4-5 дней)
1. Переделать HandleSubjects (главное меню)
2. Handlers для "Мои учителя"
3. Handlers для "Публичные учителя" (с пагинацией)
4. Handlers для "Найти учителя" (коды + заявки)
5. Handlers для "Мои заявки"

### Этап 5: Интерфейс учителя (3-4 дня)
1. Настройки учителя (публичность)
2. Управление invite-кодами
3. Просмотр и обработка заявок
4. Список студентов + отзыв доступа

### Этап 6: State handlers (1-2 дня)
1. Обработка ввода invite-кода
2. Обработка поиска учителя
3. Обработка сообщения в заявке
4. Обработка создания invite-кода

### Этап 7: Уведомления (1 день)
1. Система отправки уведомлений
2. Шаблоны сообщений

### Этап 8: Тестирование и багфиксы (2-3 дня)
1. E2E тестирование всех флоу
2. Проверка edge cases
3. Оптимизация запросов к БД
4. Исправление найденных багов

### Этап 9: Документация (1 день)
1. API документация
2. Обновление README
3. Инструкции для учителей и студентов

---

## 11. ДОПОЛНИТЕЛЬНЫЕ ФИЧИ (опционально, на будущее)

### Фаза 2:
- **Рейтинг учителей**: отзывы от студентов, средняя оценка
- **Расширенный поиск**: по предмету, цене, дням недели, времени
- **Избранные учителя**: закладки для быстрого доступа
- **Статистика для учителя**: сколько студентов активно, средняя посещаемость
- **Групповые коды**: разные коды для разных групп студентов
- **Истечение доступа**: автоматический отзыв через N дней неактивности
- **Уведомления по расписанию**: напоминания о занятиях
- **Экспорт данных**: список студентов в CSV

---

## 12. ТЕХНИЧЕСКИЕ ТРЕБОВАНИЯ

### Производительность:
- Запросы к БД должны использовать индексы
- Пагинация для больших списков (>20 элементов)
- Кеширование списка публичных учителей (Redis, если есть)

### Безопасность:
- Валидация всех входных данных
- Проверка прав доступа на каждом уровне
- SQL-инъекции: использовать prepared statements
- Rate limiting для генерации кодов (макс 10 кодов/день на учителя)

### Логирование:
- Логировать все действия с доступом (предоставление/отзыв)
- Логировать использование invite-кодов
- Логировать одобрение/отклонение заявок

### Обработка ошибок:
- Graceful degradation: если сервис недоступен, показать сообщение
- Retry логика для критичных операций
- User-friendly сообщения об ошибках

---

## 13. КРИТЕРИИ ПРИЁМКИ

✅ Учитель может переключаться между публичным/приватным статусом  
✅ Публичные учителя видны всем студентам в разделе "Публичные учителя"  
✅ Приватные учителя видны только студентам с доступом  
✅ Учитель может создавать invite-коды с ограничениями  
✅ Студент может использовать invite-код и получить доступ  
✅ Студент может отправить заявку приватному учителю  
✅ Учитель получает уведомление о новой заявке  
✅ Учитель может одобрить/отклонить заявку с сообщением  
✅ Студент получает уведомление о решении по заявке  
✅ Учитель может видеть список своих студентов  
✅ Учитель может отозвать доступ у студента  
✅ Студент видит категоризированный список: "Мои учителя" / "Публичные"  
✅ Интерфейс /subjects работает корректно с пагинацией  
✅ Кнопки предметов работают и ведут на правильные экраны  
✅ Все состояния и переходы логичны и интуитивны  

---

**ИТОГО: ~18-25 рабочих дней на полную реализацию**


