package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	repo "github.com/lucky720s/diplomaflow/internal/adapters/repository/postgres"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/internal/usecase"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// 1. Test Suite с общими ресурсами
type IntegrationTestSuite struct {
	suite.Suite
	db       *sql.DB
	pgCont   *tcpostgres.PostgresContainer
	handler  *Handler
	router   http.Handler
	dbURL    string
	testData struct {
		departmentID string
	}
}

// 2. SetupSuite — выполняется один раз перед всеми тестами
func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	log := logger.New()

	// --- Запускаем контейнер с PostgreSQL ---
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:15-alpine",
		tcpostgres.WithDatabase("test-db"),
		tcpostgres.WithUsername("user"),
		tcpostgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	s.Require().NoError(err)
	s.pgCont = pgContainer

	// Получаем connection string для созданной БД
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)
	s.dbURL = connStr

	// --- Подключаемся к БД и выполняем миграции ---
	db, err := sql.Open("postgres", connStr)
	s.Require().NoError(err)
	s.db = db

	_, filename, _, _ := runtime.Caller(0) // вернёт полный путь к handler_integration_test.go
	dir := filepath.Dir(filename)
	projectRoot := filepath.Join(dir, "../../..") // поднимаемся к корню проекта

	migrationsPath := "file://" + filepath.ToSlash(filepath.Join(projectRoot, "migrations"))

	m, err := migrate.New(migrationsPath, connStr)
	s.Require().NoError(err)

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		s.Require().NoError(err)
	}

	// --- Собираем приложение для теста ---
	studentRepo := repo.NewStudentRepo(s.db)
	departmentRepo := repo.NewDepartmentRepo(s.db)
	studentUsecase := usecase.NewStudentUsecase(studentRepo, departmentRepo, log)
	departmentUsecase := usecase.NewDepartmentUsecase(departmentRepo, log)

	s.handler = NewHandler(log, studentUsecase, departmentUsecase)
	s.router = s.handler.InitRoutes()

	// --- Заполняем базу тестовыми данными ---
	s.seedData()
}

// 3. TearDownSuite — выполняется один раз после всех тестов
func (s *IntegrationTestSuite) TearDownSuite() {
	_ = s.db.Close()
	err := s.pgCont.Terminate(context.Background())
	s.Require().NoError(err)
}

// 4. Вспомогательная функция для seed данных
func (s *IntegrationTestSuite) seedData() {
	s.testData.departmentID = "dep-cs-test"
	_, err := s.db.Exec(
		"INSERT INTO departments (id, name, university_id) VALUES ($1, $2, $3)",
		s.testData.departmentID,
		"Test CS Department",
		"uni-test",
	)
	s.Require().NoError(err)
}

// 5. Запуск Test Suite
func TestIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests: set INTEGRATION_TESTS=true to run")
	}
	suite.Run(t, new(IntegrationTestSuite))
}

// 6. Сам тест
func (s *IntegrationTestSuite) TestStudentAPI_CreateAndGet() {
	// --- Этап 1: Создание студента ---
	studentData := []byte(fmt.Sprintf(
		`{"full_name": "Интеграционный Тест", "department_id": "%s"}`,
		s.testData.departmentID,
	))

	req, err := http.NewRequest("POST", "/api/v1/students", bytes.NewBuffer(studentData))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	// Проверяем статус код
	s.Equal(http.StatusCreated, rr.Code, "Create student should return 201")

	// Декодируем ответ и сохраняем ID
	var createdStudent domain.Student
	err = json.Unmarshal(rr.Body.Bytes(), &createdStudent)
	s.Require().NoError(err)
	s.NotEmpty(createdStudent.ID, "Created student ID should not be empty")

	// --- Этап 2: Получение созданного студента ---
	getReq, err := http.NewRequest("GET", "/api/v1/students/"+createdStudent.ID, nil)
	s.Require().NoError(err)

	getRR := httptest.NewRecorder()
	s.router.ServeHTTP(getRR, getReq)

	// Проверяем статус код
	s.Equal(http.StatusOK, getRR.Code, "Get student should return 200")

	// Декодируем ответ и проверяем данные
	var fetchedStudent domain.StudentWithDepartment
	err = json.Unmarshal(getRR.Body.Bytes(), &fetchedStudent)
	s.Require().NoError(err)

	s.Equal("Интеграционный Тест", fetchedStudent.FullName)
	s.Equal(s.testData.departmentID, fetchedStudent.Department.ID)
	s.Equal("Test CS Department", fetchedStudent.Department.Name)
}
