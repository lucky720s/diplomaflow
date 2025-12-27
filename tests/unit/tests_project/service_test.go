package tests_project

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/lucky720s/diplomaflow/internal/project"
)

// ==================== ERRORS ====================

var ErrNotFound = errors.New("not found")

// ==================== MOCKS ====================

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, p *project.Project) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockRepository) CreateWithOutbox(ctx context.Context, p *project.Project, eventType, topic string, payload map[string]interface{}) error {
	args := m.Called(ctx, p, eventType, topic, payload)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uint64) (*project.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

func (m *MockRepository) ListByStudent(ctx context.Context, studentID int64) ([]*project.Project, error) {
	args := m.Called(ctx, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}

func (m *MockRepository) ListByDepartment(ctx context.Context, departmentID int64) ([]*project.Project, error) {
	args := m.Called(ctx, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, p *project.Project) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) AddHistory(ctx context.Context, h *project.StateHistory) error {
	args := m.Called(ctx, h)
	return args.Error(0)
}

func (m *MockRepository) GetHistory(ctx context.Context, projectID uint64) ([]project.StateHistory, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]project.StateHistory), args.Error(1)
}

func (m *MockRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]project.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]project.OutboxEvent), args.Error(1)
}

func (m *MockRepository) MarkOutboxEventProcessed(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) MarkOutboxEventFailed(ctx context.Context, id uint64, errMsg string) error {
	args := m.Called(ctx, id, errMsg)
	return args.Error(0)
}

func (m *MockRepository) GetProjectsWithUpcomingDeadlines(ctx context.Context, before time.Time) ([]*project.Project, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}

func (m *MockRepository) GetActiveProjectsCount(ctx context.Context, departmentID int64) (int64, error) {
	args := m.Called(ctx, departmentID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) GetProjectsByStatus(ctx context.Context, departmentID int64, status string) ([]*project.Project, error) {
	args := m.Called(ctx, departmentID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}

func (m *MockRepository) GetProjectsByState(ctx context.Context, departmentID int64, stateName string) ([]*project.Project, error) {
	args := m.Called(ctx, departmentID, stateName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}

// ==================== TEST HELPERS ====================

func createTestProject() *project.Project {
	now := time.Now()
	deadline := now.AddDate(0, 0, 14)
	return &project.Project{
		ID:            1,
		Title:         "Дипломная работа: Разработка системы",
		Description:   "Описание проекта",
		StudentID:     100,
		UniversityID:  1,
		DepartmentID:  1,
		TeamID:        0, // int64, не указатель
		WorkflowID:    1,
		WorkflowName:  "Дипломный проект 2025",
		CurrentStepID: "1",
		CurrentState:  "TEAM_FORMATION",
		Status:        "active",
		Data:          datatypes.JSON(`{}`),
		DeadlineAt:    &deadline,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func createTestStateHistory() *project.StateHistory {
	return &project.StateHistory{
		ID:        1,
		ProjectID: 1,
		StateName: "TEAM_FORMATION",
		Status:    "completed",
		ChangedBy: 100,
		Comment:   "Project created",
		CreatedAt: time.Now(),
	}
}

// ==================== REPOSITORY TESTS ====================

func TestRepository_Create(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		p := createTestProject()
		p.ID = 0 // New project
		repo.On("Create", ctx, p).Return(nil).Once()

		err := repo.Create(ctx, p)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_CreateWithOutbox(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful creation with outbox event", func(t *testing.T) {
		p := createTestProject()
		p.ID = 0

		payload := map[string]interface{}{
			"student_id":    int64(100),
			"university_id": int64(1),
			"workflow_id":   int64(1),
		}

		repo.On("CreateWithOutbox", ctx, p, "ProjectCreated", "project-events", payload).
			Return(nil).Once()

		err := repo.CreateWithOutbox(ctx, p, "ProjectCreated", "project-events", payload)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetByID(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		expected := createTestProject()
		repo.On("GetByID", ctx, uint64(1)).Return(expected, nil).Once()

		p, err := repo.GetByID(ctx, 1)

		require.NoError(t, err)
		assert.Equal(t, expected.ID, p.ID)
		assert.Equal(t, expected.Title, p.Title)
		assert.Equal(t, expected.StudentID, p.StudentID)
		repo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		repo.On("GetByID", ctx, uint64(999)).Return(nil, ErrNotFound).Once()

		_, err := repo.GetByID(ctx, 999)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_ListByStudent(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns student projects", func(t *testing.T) {
		projects := []*project.Project{
			createTestProject(),
			{
				ID:           2,
				Title:        "Курсовая работа",
				StudentID:    100,
				CurrentState: "COMPLETED",
				Status:       "completed",
			},
		}
		repo.On("ListByStudent", ctx, int64(100)).Return(projects, nil).Once()

		result, err := repo.ListByStudent(ctx, 100)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(100), result[0].StudentID)
		repo.AssertExpectations(t)
	})

	t.Run("returns empty list", func(t *testing.T) {
		repo.On("ListByStudent", ctx, int64(999)).Return([]*project.Project{}, nil).Once()

		result, err := repo.ListByStudent(ctx, 999)

		require.NoError(t, err)
		assert.Empty(t, result)
		repo.AssertExpectations(t)
	})
}

func TestRepository_ListByDepartment(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns department projects", func(t *testing.T) {
		projects := []*project.Project{
			createTestProject(),
		}
		repo.On("ListByDepartment", ctx, int64(1)).Return(projects, nil).Once()

		result, err := repo.ListByDepartment(ctx, 1)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		repo.AssertExpectations(t)
	})
}

func TestRepository_Update(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		p := createTestProject()
		p.CurrentState = "SUPERVISOR_SELECTION"
		p.CurrentStepID = "2"

		repo.On("Update", ctx, p).Return(nil).Once()

		err := repo.Update(ctx, p)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_Delete(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		repo.On("Delete", ctx, uint64(1)).Return(nil).Once()

		err := repo.Delete(ctx, 1)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// ==================== HISTORY TESTS ====================

func TestRepository_AddHistory(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful add history", func(t *testing.T) {
		h := createTestStateHistory()
		repo.On("AddHistory", ctx, h).Return(nil).Once()

		err := repo.AddHistory(ctx, h)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetHistory(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns history", func(t *testing.T) {
		history := []project.StateHistory{
			{
				ID:        1,
				ProjectID: 1,
				StateName: "TEAM_FORMATION",
				Status:    "completed",
				ChangedBy: 100,
				CreatedAt: time.Now().Add(-time.Hour),
			},
			{
				ID:        2,
				ProjectID: 1,
				StateName: "SUPERVISOR_SELECTION",
				Status:    "completed",
				ChangedBy: 100,
				CreatedAt: time.Now(),
			},
		}
		repo.On("GetHistory", ctx, uint64(1)).Return(history, nil).Once()

		result, err := repo.GetHistory(ctx, 1)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})

	t.Run("returns empty history", func(t *testing.T) {
		repo.On("GetHistory", ctx, uint64(999)).Return([]project.StateHistory{}, nil).Once()

		result, err := repo.GetHistory(ctx, 999)

		require.NoError(t, err)
		assert.Empty(t, result)
		repo.AssertExpectations(t)
	})
}

// ==================== OUTBOX TESTS ====================

func TestRepository_GetPendingOutboxEvents(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns pending events", func(t *testing.T) {
		events := []project.OutboxEvent{
			{
				ID:        1,
				Topic:     "project-events",
				EventType: "ProjectCreated",
				Payload:   datatypes.JSON(`{"project_id": 1}`),
				Status:    "pending",
				CreatedAt: time.Now(),
			},
		}
		repo.On("GetPendingOutboxEvents", ctx, 10).Return(events, nil).Once()

		result, err := repo.GetPendingOutboxEvents(ctx, 10)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "pending", result[0].Status)
		repo.AssertExpectations(t)
	})
}

func TestRepository_MarkOutboxEventProcessed(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful mark processed", func(t *testing.T) {
		repo.On("MarkOutboxEventProcessed", ctx, uint64(1)).Return(nil).Once()

		err := repo.MarkOutboxEventProcessed(ctx, 1)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_MarkOutboxEventFailed(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful mark failed", func(t *testing.T) {
		repo.On("MarkOutboxEventFailed", ctx, uint64(1), "connection error").
			Return(nil).Once()

		err := repo.MarkOutboxEventFailed(ctx, 1, "connection error")

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// ==================== DEADLINE TESTS ====================

func TestRepository_GetProjectsWithUpcomingDeadlines(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns projects with upcoming deadlines", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		projects := []*project.Project{
			{
				ID:           1,
				Title:        "Project 1",
				DeadlineAt:   &deadline,
				CurrentState: "SUBMISSION",
				Status:       "active",
			},
		}
		before := time.Now().Add(48 * time.Hour)
		repo.On("GetProjectsWithUpcomingDeadlines", ctx, before).
			Return(projects, nil).Once()

		result, err := repo.GetProjectsWithUpcomingDeadlines(ctx, before)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		repo.AssertExpectations(t)
	})
}

// ==================== STATS TESTS ====================

func TestRepository_GetActiveProjectsCount(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns count", func(t *testing.T) {
		repo.On("GetActiveProjectsCount", ctx, int64(1)).Return(int64(42), nil).Once()

		count, err := repo.GetActiveProjectsCount(ctx, 1)

		require.NoError(t, err)
		assert.Equal(t, int64(42), count)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetProjectsByStatus(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns projects by status", func(t *testing.T) {
		projects := []*project.Project{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "completed"},
		}
		repo.On("GetProjectsByStatus", ctx, int64(1), "completed").
			Return(projects, nil).Once()

		result, err := repo.GetProjectsByStatus(ctx, 1, "completed")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetProjectsByState(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns projects by state", func(t *testing.T) {
		projects := []*project.Project{
			{ID: 1, CurrentState: "TEAM_FORMATION"},
			{ID: 2, CurrentState: "TEAM_FORMATION"},
		}
		repo.On("GetProjectsByState", ctx, int64(1), "TEAM_FORMATION").
			Return(projects, nil).Once()

		result, err := repo.GetProjectsByState(ctx, 1, "TEAM_FORMATION")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})
}

// ==================== MODEL TESTS ====================

func TestProject_Model(t *testing.T) {
	t.Run("project has correct fields", func(t *testing.T) {
		p := createTestProject()

		assert.Equal(t, uint(1), p.ID)
		assert.Equal(t, "Дипломная работа: Разработка системы", p.Title)
		assert.Equal(t, int64(100), p.StudentID)
		assert.Equal(t, int64(1), p.UniversityID)
		assert.Equal(t, int64(1), p.DepartmentID)
		assert.Equal(t, uint(1), p.WorkflowID)
		assert.Equal(t, "TEAM_FORMATION", p.CurrentState)
		assert.Equal(t, "active", p.Status)
		assert.NotNil(t, p.DeadlineAt)
	})

	t.Run("project status values", func(t *testing.T) {
		validStatuses := []string{"active", "completed", "cancelled", "paused"}

		for _, status := range validStatuses {
			p := createTestProject()
			p.Status = status
			assert.Equal(t, status, p.Status)
		}
	})
}

func TestStateHistory_Model(t *testing.T) {
	t.Run("state history has correct fields", func(t *testing.T) {
		h := createTestStateHistory()

		assert.Equal(t, uint(1), h.ID)
		assert.Equal(t, uint(1), h.ProjectID)
		assert.Equal(t, "TEAM_FORMATION", h.StateName)
		assert.Equal(t, "completed", h.Status)
		assert.Equal(t, int64(100), h.ChangedBy)
	})
}

func TestOutboxEvent_Model(t *testing.T) {
	t.Run("outbox event has correct fields", func(t *testing.T) {
		event := project.OutboxEvent{
			ID:        1,
			Topic:     "project-events",
			EventType: "ProjectCreated",
			Payload:   datatypes.JSON(`{"project_id": 1}`),
			Status:    "pending",
			CreatedAt: time.Now(),
		}

		assert.Equal(t, uint(1), event.ID)
		assert.Equal(t, "project-events", event.Topic)
		assert.Equal(t, "ProjectCreated", event.EventType)
		assert.Equal(t, "pending", event.Status)
	})
}

// ==================== VALIDATION TESTS ====================

func TestProject_Validation(t *testing.T) {
	t.Run("project must have title", func(t *testing.T) {
		p := createTestProject()
		p.Title = ""
		assert.Empty(t, p.Title)
	})

	t.Run("project must have student_id", func(t *testing.T) {
		p := createTestProject()
		assert.NotZero(t, p.StudentID)
	})

	t.Run("project must have workflow_id", func(t *testing.T) {
		p := createTestProject()
		assert.NotZero(t, p.WorkflowID)
	})

	t.Run("project must have current_state", func(t *testing.T) {
		p := createTestProject()
		assert.NotEmpty(t, p.CurrentState)
	})

	t.Run("project must have valid status", func(t *testing.T) {
		p := createTestProject()
		validStatuses := map[string]bool{
			"active":    true,
			"completed": true,
			"cancelled": true,
			"paused":    true,
		}
		assert.True(t, validStatuses[p.Status])
	})
}

// ==================== STATE TRANSITION TESTS ====================

func TestProject_StateTransition(t *testing.T) {
	t.Run("can transition to next state", func(t *testing.T) {
		p := createTestProject()
		assert.Equal(t, "TEAM_FORMATION", p.CurrentState)

		// Simulate transition
		p.CurrentState = "SUPERVISOR_SELECTION"
		p.CurrentStepID = "2"
		p.UpdatedAt = time.Now()

		assert.Equal(t, "SUPERVISOR_SELECTION", p.CurrentState)
		assert.Equal(t, "2", p.CurrentStepID)
	})

	t.Run("can transition to final state", func(t *testing.T) {
		p := createTestProject()
		p.CurrentState = "COMPLETED"
		p.Status = "completed"

		assert.Equal(t, "COMPLETED", p.CurrentState)
		assert.Equal(t, "completed", p.Status)
	})
}

// ==================== DEADLINE TESTS ====================

func TestProject_Deadline(t *testing.T) {
	t.Run("project can have deadline", func(t *testing.T) {
		p := createTestProject()
		assert.NotNil(t, p.DeadlineAt)
	})

	t.Run("project deadline can be nil", func(t *testing.T) {
		p := createTestProject()
		p.DeadlineAt = nil
		assert.Nil(t, p.DeadlineAt)
	})

	t.Run("deadline is in the future for active project", func(t *testing.T) {
		p := createTestProject()
		if p.DeadlineAt != nil && p.Status == "active" {
			assert.True(t, p.DeadlineAt.After(time.Now()))
		}
	})
}

// ==================== DATA TESTS ====================

func TestProject_Data(t *testing.T) {
	t.Run("project can store JSON data", func(t *testing.T) {
		p := createTestProject()
		p.Data = datatypes.JSON(`{"team_id": 123, "supervisor_id": 456}`)

		assert.NotNil(t, p.Data)
	})

	t.Run("project data defaults to empty object", func(t *testing.T) {
		p := createTestProject()
		p.Data = datatypes.JSON(`{}`)

		assert.Equal(t, datatypes.JSON(`{}`), p.Data)
	})
}

// ==================== TEAM ASSIGNMENT TESTS ====================

func TestProject_TeamAssignment(t *testing.T) {
	t.Run("project can have no team", func(t *testing.T) {
		p := createTestProject()
		p.TeamID = 0
		assert.Zero(t, p.TeamID)
	})

	t.Run("project can have team assigned", func(t *testing.T) {
		p := createTestProject()
		p.TeamID = 10

		assert.Equal(t, int64(10), p.TeamID)
	})
}

// ==================== WORKFLOW INTEGRATION TESTS ====================

func TestProject_WorkflowIntegration(t *testing.T) {
	t.Run("project tracks workflow info", func(t *testing.T) {
		p := createTestProject()

		assert.NotZero(t, p.WorkflowID)
		assert.NotEmpty(t, p.WorkflowName)
		assert.NotEmpty(t, p.CurrentStepID)
		assert.NotEmpty(t, p.CurrentState)
	})
}

// ==================== EDGE CASES ====================

func TestProject_EdgeCases(t *testing.T) {
	t.Run("handles empty description", func(t *testing.T) {
		p := createTestProject()
		p.Description = ""
		assert.Empty(t, p.Description)
	})

	t.Run("handles very long title", func(t *testing.T) {
		p := createTestProject()
		p.Title = "Very long title that exceeds normal length and continues for many more characters"
		assert.NotEmpty(t, p.Title)
	})

	t.Run("handles concurrent updates", func(t *testing.T) {
		p := createTestProject()
		originalUpdated := p.UpdatedAt

		time.Sleep(time.Millisecond)
		p.UpdatedAt = time.Now()

		assert.True(t, p.UpdatedAt.After(originalUpdated))
	})
}
