package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type DepartmentConfigService struct {
	repo   *DepartmentConfigRepository
	wfRepo Repository
	logger *zap.Logger
}

func NewDepartmentConfigService(repo *DepartmentConfigRepository, wfRepo Repository, logger *zap.Logger) *DepartmentConfigService {
	return &DepartmentConfigService{
		repo:   repo,
		wfRepo: wfRepo,
		logger: logger,
	}
}

// CreateConfig создаёт конфигурацию workflow для кафедры
func (s *DepartmentConfigService) CreateConfig(ctx context.Context, input *CreateDepartmentConfigInput) (*DepartmentWorkflowConfig, error) {
	// Проверяем что workflow существует
	wf, err := s.wfRepo.GetWorkflow(ctx, input.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	teamSettingsJSON, _ := json.Marshal(input.TeamSettings)
	deadlineOverridesJSON, _ := json.Marshal(input.DeadlineOverrides)
	configOverridesJSON, _ := json.Marshal(input.ConfigOverrides)

	config := &DepartmentWorkflowConfig{
		DepartmentID:      input.DepartmentID,
		WorkflowID:        wf.ID,
		AcademicYear:      input.AcademicYear,
		IsActive:          false,
		TeamSettings:      datatypes.JSON(teamSettingsJSON),
		DeadlineOverrides: datatypes.JSON(deadlineOverridesJSON),
		ConfigOverrides:   datatypes.JSON(configOverridesJSON),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.Create(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

// GetEffectiveWorkflow возвращает workflow с применёнными конфигурациями кафедры
func (s *DepartmentConfigService) GetEffectiveWorkflow(ctx context.Context, departmentID int64, academicYear string) (*EffectiveWorkflow, error) {
	// Получаем активную конфигурацию
	config, err := s.repo.GetActiveConfig(ctx, departmentID, academicYear)
	if err != nil {
		// Fallback на дефолтный workflow кафедры
		wf, err := s.wfRepo.GetActiveWorkflowByDepartment(ctx, departmentID)
		if err != nil {
			return nil, err
		}
		return s.workflowToEffective(wf, nil), nil
	}

	// Получаем базовый workflow
	wf, wfErr := s.wfRepo.GetActiveWorkflowByDepartment(ctx, departmentID)
	if wfErr != nil {
		s.logger.Warn("Failed to get active workflow", zap.Error(wfErr))
		return nil, fmt.Errorf("failed to get workflow: %w", wfErr)
	}

	// Применяем конфигурации
	effective := s.workflowToEffective(wf, config)

	// Добавляем кастомные этапы
	customSteps, _ := s.repo.GetCustomSteps(ctx, config.ID)
	effective.CustomSteps = customSteps

	return effective, nil
}

// EffectiveWorkflow - workflow с применёнными настройками кафедры
type EffectiveWorkflow struct {
	Workflow     *Workflow
	TeamSettings *TeamSettings
	States       []EffectiveState
	CustomSteps  []DepartmentCustomStep
	Transitions  []Transition
}

type EffectiveState struct {
	State
	EffectiveDeadline *time.Time
	OverriddenConfig  map[string]interface{}
}

func (s *DepartmentConfigService) workflowToEffective(wf *Workflow, config *DepartmentWorkflowConfig) *EffectiveWorkflow {
	effective := &EffectiveWorkflow{
		Workflow:    wf,
		Transitions: wf.Transitions,
	}

	// Парсим настройки команды
	if config != nil && len(config.TeamSettings) > 0 {
		var ts TeamSettings
		if err := json.Unmarshal(config.TeamSettings, &ts); err != nil {
			s.logger.Warn("Failed to unmarshal team settings", zap.Error(err))
		}
		effective.TeamSettings = &ts
	} else {
		// Дефолтные настройки из workflow
		var wfSettings WorkflowSettings
		if err := json.Unmarshal(wf.Settings, &wfSettings); err != nil {
			s.logger.Warn("Failed to unmarshal workflow settings", zap.Error(err))
		}
		effective.TeamSettings = &TeamSettings{
			AllowSolo: wfSettings.AllowSoloProject,
			MinSize:   wfSettings.MinTeamSize,
			MaxSize:   wfSettings.MaxTeamSize,
		}
	}

	// Применяем переопределения дедлайнов
	var deadlineOverrides []DeadlineOverride
	if config != nil {
		if err := json.Unmarshal(config.DeadlineOverrides, &deadlineOverrides); err != nil {
			s.logger.Warn("Failed to unmarshal deadline overrides", zap.Error(err))
		}
	}

	deadlineMap := make(map[int64]DeadlineOverride)
	for _, do := range deadlineOverrides {
		deadlineMap[do.StateID] = do
	}

	// Формируем effective states
	for _, state := range wf.States {
		es := EffectiveState{State: state}

		if override, ok := deadlineMap[state.ID]; ok {
			if override.FixedDate != nil {
				es.EffectiveDeadline = override.FixedDate
			} else if override.DurationDays != nil {
				// Рассчитываем от начала workflow или предыдущего этапа
				// (упрощённо - от текущей даты)
				deadline := time.Now().AddDate(0, 0, *override.DurationDays)
				es.EffectiveDeadline = &deadline
			}
		}

		effective.States = append(effective.States, es)
	}

	return effective
}

// AddCustomStep добавляет кастомный этап для кафедры
func (s *DepartmentConfigService) AddCustomStep(ctx context.Context, configID int64, input *AddCustomStepInput) (*DepartmentCustomStep, error) {
	configJSON, _ := json.Marshal(input.Config)

	step := &DepartmentCustomStep{
		DepartmentConfigID: configID,
		Name:               input.Name,
		DisplayName:        input.DisplayName,
		StepType:           input.StepType,
		InsertAfterStateID: input.InsertAfterStateID,
		Config:             datatypes.JSON(configJSON),
		IsRequired:         input.IsRequired,
		DurationDays:       input.DurationDays,
		CreatedAt:          time.Now(),
	}

	if err := s.repo.CreateCustomStep(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// ActivateConfig активирует конфигурацию
func (s *DepartmentConfigService) ActivateConfig(ctx context.Context, configID int64) error {
	config, err := s.repo.Get(ctx, configID)
	if err != nil {
		return err
	}

	// Деактивируем все другие конфиги этой кафедры для этого года
	if err := s.repo.DeactivateAll(ctx, config.DepartmentID, config.AcademicYear); err != nil {
		return err
	}

	config.IsActive = true
	config.UpdatedAt = time.Now()

	return s.repo.Update(ctx, config)
}

// Input structs
type CreateDepartmentConfigInput struct {
	DepartmentID      int64
	WorkflowID        int64
	AcademicYear      string
	TeamSettings      *TeamSettings
	DeadlineOverrides []DeadlineOverride
	ConfigOverrides   map[string]interface{}
}

type AddCustomStepInput struct {
	Name               string
	DisplayName        string
	StepType           string
	InsertAfterStateID *int64
	Config             map[string]interface{}
	IsRequired         bool
	DurationDays       int32
}
