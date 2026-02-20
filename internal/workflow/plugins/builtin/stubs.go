package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	godocx "github.com/lukasjarosch/go-docx"
)

// =============== Email Plugin ===============
type EmailPlugin struct{}

func NewEmailPlugin() *EmailPlugin { return &EmailPlugin{} }

func (p *EmailPlugin) ID() string   { return "SEND_EMAIL" }
func (p *EmailPlugin) Name() string { return "Отправить email" }
func (p *EmailPlugin) Description() string {
	return "Отправляет email пользователям"
}
func (p *EmailPlugin) Category() string   { return plugins.CategoryNotification }
func (p *EmailPlugin) IsReversible() bool { return false }

func (p *EmailPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["subject", "body"],
		"properties": {
			"subject": {"type": "string", "title": "Тема письма"},
			"body": {"type": "string", "title": "Текст письма"},
			"recipients": {"type": "string", "enum": ["student", "team", "supervisor"]}
		}
	}`)
}

func (p *EmailPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *EmailPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"status": "stub",
			"reason": "email plugin not yet implemented",
		},
	}
}
func (p *EmailPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }

// =============== Reminder Plugin ===============
type ReminderPlugin struct{}

func NewReminderPlugin() *ReminderPlugin { return &ReminderPlugin{} }

func (p *ReminderPlugin) ID() string   { return "SCHEDULE_REMINDER" }
func (p *ReminderPlugin) Name() string { return "Запланировать напоминание" }
func (p *ReminderPlugin) Description() string {
	return "Создаёт отложенное напоминание"
}
func (p *ReminderPlugin) Category() string   { return plugins.CategoryNotification }
func (p *ReminderPlugin) IsReversible() bool { return true }

func (p *ReminderPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"days_before_deadline": {"type": "integer", "default": 3},
			"message": {"type": "string"}
		}
	}`)
}

func (p *ReminderPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *ReminderPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"status": "stub",
			"reason": "reminder plugin not yet implemented",
		},
	}
}
func (p *ReminderPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }

// =============== Turnitin Plugin ===============
type TurnitinPlugin struct{}

func NewTurnitinPlugin() *TurnitinPlugin { return &TurnitinPlugin{} }

func (p *TurnitinPlugin) ID() string   { return "CHECK_TURNITIN" }
func (p *TurnitinPlugin) Name() string { return "Проверка Turnitin" }
func (p *TurnitinPlugin) Description() string {
	return "Отправляет документ на проверку в Turnitin"
}
func (p *TurnitinPlugin) Category() string   { return plugins.CategoryExternal }
func (p *TurnitinPlugin) IsReversible() bool { return false }

func (p *TurnitinPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"api_key": {"type": "string"},
			"min_score": {"type": "number", "default": 70}
		}
	}`)
}

func (p *TurnitinPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *TurnitinPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"status": "stub",
			"reason": "turnitin plugin not yet implemented",
		},
	}
}
func (p *TurnitinPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }

// =============== File Validation Plugin ===============
type FileValidationPlugin struct{}

func NewFileValidationPlugin() *FileValidationPlugin { return &FileValidationPlugin{} }

func (p *FileValidationPlugin) ID() string   { return "VALIDATE_FILES" }
func (p *FileValidationPlugin) Name() string { return "Валидация файлов" }
func (p *FileValidationPlugin) Description() string {
	return "Проверяет наличие и корректность загруженных файлов"
}
func (p *FileValidationPlugin) Category() string   { return plugins.CategoryValidation }
func (p *FileValidationPlugin) IsReversible() bool { return false }

func (p *FileValidationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"min_files": {"type": "integer", "default": 1, "title": "Минимум файлов"},
			"max_size_mb": {"type": "integer", "default": 50, "title": "Макс размер (МБ)"},
			"allowed_extensions": {
				"type": "array",
				"items": {"type": "string"},
				"default": [".pdf", ".docx", ".doc"]
			}
		}
	}`)
}

func (p *FileValidationPlugin) Validate(config map[string]interface{}) error { return nil }

func (p *FileValidationPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	// Проверяем наличие файлов в payload или project_data
	minFiles := int64(1)
	if mf, ok := actx.Config["min_files"].(float64); ok && mf > 0 {
		minFiles = int64(mf)
	}

	// Проверяем file_ids в payload
	var fileIDs []interface{}
	if actx.Payload != nil {
		if ids, ok := actx.Payload["file_ids"].([]interface{}); ok {
			fileIDs = ids
		}
	}

	// Также проверяем в project_data
	if len(fileIDs) == 0 {
		stateKey := fmt.Sprintf("files_state_%d", actx.StateID)
		if ids, ok := actx.ProjectData[stateKey].([]interface{}); ok {
			fileIDs = ids
		}
	}

	if int64(len(fileIDs)) < minFiles {
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("требуется минимум %d файл(ов), загружено: %d", minFiles, len(fileIDs)),
		}
	}

	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"validated":   true,
			"files_count": len(fileIDs),
		},
	}
}

func (p *FileValidationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// =============== Form Validation Plugin ===============
type FormValidationPlugin struct{}

func NewFormValidationPlugin() *FormValidationPlugin { return &FormValidationPlugin{} }

func (p *FormValidationPlugin) ID() string   { return "VALIDATE_DATA" }
func (p *FormValidationPlugin) Name() string { return "Валидация формы" }
func (p *FormValidationPlugin) Description() string {
	return "Проверяет, что форма была заполнена для текущего этапа"
}
func (p *FormValidationPlugin) Category() string   { return plugins.CategoryValidation }
func (p *FormValidationPlugin) IsReversible() bool { return false }

func (p *FormValidationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"require_all_fields": {"type": "boolean", "default": true},
			"form_id": {"type": "string", "title": "ID формы (опционально)"}
		}
	}`)
}

func (p *FormValidationPlugin) Validate(config map[string]interface{}) error { return nil }

func (p *FormValidationPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	// Проверяем наличие form_submission в project_data
	// form_service при SubmitForm сохраняет данные в form_submissions таблицу
	// Фронтенд должен сначала вызвать SubmitForm, потом PerformAction
	// Плагин проверяет что в project_data есть маркер заполненной формы

	projectID := actx.ProjectID
	stateID := actx.StateID

	// Вариант 1: Проверяем через project_data (если form_service пишет туда)
	formKey := fmt.Sprintf("form_submitted_state_%d", stateID)
	if val, ok := actx.ProjectData[formKey]; ok {
		if submitted, ok := val.(bool); ok && submitted {
			return &plugins.ActionResult{
				Success: true,
				Data: map[string]interface{}{
					"validated":  true,
					"project_id": projectID,
					"state_id":   stateID,
				},
			}
		}
	}

	// Вариант 2: Проверяем наличие любого form_submission_id в данных
	formSubmissionKey := fmt.Sprintf("form_submission_id_state_%d", stateID)
	if submissionID, ok := actx.ProjectData[formSubmissionKey]; ok && submissionID != nil && submissionID != "" {
		return &plugins.ActionResult{
			Success: true,
			Data: map[string]interface{}{
				"validated":     true,
				"submission_id": submissionID,
			},
		}
	}

	// Вариант 3: Проверяем через payload (фронт передаёт submission_id в payload при PerformAction)
	if actx.Payload != nil {
		if submissionID, ok := actx.Payload["form_submission_id"]; ok && submissionID != nil {
			return &plugins.ActionResult{
				Success: true,
				Data: map[string]interface{}{
					"validated":     true,
					"submission_id": submissionID,
					"source":        "payload",
				},
			}
		}
	}

	return &plugins.ActionResult{
		Success: false,
		Error:   fmt.Errorf("форма не заполнена для этапа %d проекта %d. Сначала заполните и отправьте форму", stateID, projectID),
	}
}

func (p *FormValidationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// =============== Grade Calculation Plugin ===============
type GradeCalculationPlugin struct{}

func NewGradeCalculationPlugin() *GradeCalculationPlugin { return &GradeCalculationPlugin{} }

func (p *GradeCalculationPlugin) ID() string   { return "CALCULATE_GRADE" }
func (p *GradeCalculationPlugin) Name() string { return "Расчёт итоговой оценки" }
func (p *GradeCalculationPlugin) Description() string {
	return "Автоматически рассчитывает итоговую оценку на основе оценок за этапы"
}
func (p *GradeCalculationPlugin) Category() string   { return plugins.CategoryGrading }
func (p *GradeCalculationPlugin) IsReversible() bool { return false }

func (p *GradeCalculationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"weights": {
				"type": "object",
				"title": "Веса оценок",
				"description": "Ключ - имя этапа, значение - вес (0.0-1.0)",
				"default": {
					"norm_control": 0.10,
					"supervisor_review": 0.10,
					"pre_defense": 0.20,
					"defense": 0.60
				}
			},
			"passing_grade": {
				"type": "integer",
				"title": "Проходной балл",
				"default": 50
			},
			"rounding": {
				"type": "string",
				"enum": ["round", "floor", "ceil"],
				"default": "round"
			}
		}
	}`)
}

func (p *GradeCalculationPlugin) Validate(config map[string]interface{}) error {
	return nil
}

func (p *GradeCalculationPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	// Получаем веса из конфига
	weights := map[string]float64{
		"norm_control":      0.10,
		"supervisor_review": 0.10,
		"pre_defense":       0.20,
		"defense":           0.60,
	}

	if w, ok := actx.Config["weights"].(map[string]interface{}); ok {
		for k, v := range w {
			if fv, ok := v.(float64); ok {
				weights[k] = fv
			}
		}
	}

	passingGrade := 50.0
	if pg, ok := actx.Config["passing_grade"].(float64); ok {
		passingGrade = pg
	}

	// Собираем оценки из project_data
	// Оценки хранятся в project_data.grades как map[step_name]grade
	grades := map[string]float64{}
	if gradesRaw, ok := actx.ProjectData["grades"].(map[string]interface{}); ok {
		for k, v := range gradesRaw {
			switch gv := v.(type) {
			case float64:
				grades[k] = gv
			case int:
				grades[k] = float64(gv)
			case int64:
				grades[k] = float64(gv)
			}
		}
	}

	// Рассчитываем взвешенную оценку
	var totalScore float64
	var totalWeight float64
	gradeDetails := map[string]interface{}{}

	for stepName, weight := range weights {
		if grade, ok := grades[stepName]; ok {
			weightedGrade := grade * weight
			totalScore += weightedGrade
			totalWeight += weight
			gradeDetails[stepName] = map[string]interface{}{
				"grade":          grade,
				"weight":         weight,
				"weighted_grade": weightedGrade,
			}
		}
	}

	var finalGrade float64
	if totalWeight > 0 {
		finalGrade = totalScore / totalWeight
	}

	// Округление
	rounding := "round"
	if r, ok := actx.Config["rounding"].(string); ok {
		rounding = r
	}

	var finalGradeInt int32
	switch rounding {
	case "floor":
		finalGradeInt = int32(finalGrade)
	case "ceil":
		finalGradeInt = int32(finalGrade + 0.999)
	default: // round
		finalGradeInt = int32(finalGrade + 0.5)
	}

	passed := float64(finalGradeInt) >= passingGrade

	// Определяем буквенную оценку
	letterGrade := "F"
	switch {
	case finalGradeInt >= 95:
		letterGrade = "A"
	case finalGradeInt >= 90:
		letterGrade = "A-"
	case finalGradeInt >= 85:
		letterGrade = "B+"
	case finalGradeInt >= 80:
		letterGrade = "B"
	case finalGradeInt >= 75:
		letterGrade = "B-"
	case finalGradeInt >= 70:
		letterGrade = "C+"
	case finalGradeInt >= 65:
		letterGrade = "C"
	case finalGradeInt >= 60:
		letterGrade = "C-"
	case finalGradeInt >= 55:
		letterGrade = "D+"
	case finalGradeInt >= 50:
		letterGrade = "D"
	}

	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"final_grade":       finalGradeInt,
			"final_grade_exact": finalGrade,
			"letter_grade":      letterGrade,
			"passed":            passed,
			"passing_grade":     passingGrade,
			"total_weight_used": totalWeight,
			"grade_details":     gradeDetails,
			"calculated_at":     time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func (p *GradeCalculationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// =============== Document Generator Plugin ===============

type DocumentGeneratorPlugin struct{}

func NewDocumentGeneratorPlugin() *DocumentGeneratorPlugin { return &DocumentGeneratorPlugin{} }

func (p *DocumentGeneratorPlugin) ID() string   { return "GENERATE_DOCUMENT" }
func (p *DocumentGeneratorPlugin) Name() string { return "Генерация документа" }
func (p *DocumentGeneratorPlugin) Description() string {
	return "Генерирует документ из шаблона .docx с подстановкой данных проекта"
}
func (p *DocumentGeneratorPlugin) Category() string   { return plugins.CategoryDocument }
func (p *DocumentGeneratorPlugin) IsReversible() bool { return false }

func (p *DocumentGeneratorPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"template_name": {
				"type": "string",
				"title": "Имя шаблона (без .docx)",
				"description": "Файл шаблона из папки templates/docx/"
			},
			"output_filename": {
				"type": "string",
				"title": "Имя выходного файла",
				"default": "generated_document.docx"
			},
			"templates_dir": {
				"type": "string",
				"title": "Путь к папке шаблонов",
				"default": "templates/docx"
			},
			"output_dir": {
				"type": "string",
				"title": "Путь к папке выходных файлов",
				"default": "/tmp/diplomaflow/generated"
			},
			"placeholders": {
				"type": "object",
				"title": "Дополнительные плейсхолдеры",
				"description": "Ключ = плейсхолдер в шаблоне, значение = ключ из project_data"
			}
		},
		"required": ["template_name"]
	}`)
}

func (p *DocumentGeneratorPlugin) Validate(config map[string]interface{}) error {
	if _, ok := config["template_name"]; !ok {
		return fmt.Errorf("template_name is required")
	}
	return nil
}

func (p *DocumentGeneratorPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	templateName, _ := actx.Config["template_name"].(string)
	if templateName == "" {
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("template_name is not configured"),
		}
	}

	outputFilename := "generated_document.docx"
	if of, ok := actx.Config["output_filename"].(string); ok && of != "" {
		outputFilename = of
	}

	templatesDir := "templates/docx"
	if td, ok := actx.Config["templates_dir"].(string); ok && td != "" {
		templatesDir = td
	}

	outputDir := "/tmp/diplomaflow/generated"
	if od, ok := actx.Config["output_dir"].(string); ok && od != "" {
		outputDir = od
	}

	// ===== Собираем replacements =====
	replacements := godocx.PlaceholderMap{
		"ProjectID":    fmt.Sprintf("%d", actx.ProjectID),
		"StateID":      fmt.Sprintf("%d", actx.StateID),
		"Date":         time.Now().Format("02.01.2006"),
		"DateTime":     time.Now().Format("02.01.2006 15:04"),
		"AcademicYear": getCurrentAcademicYear(),
	}

	// Данные из project_data
	projectDataFields := map[string]string{
		"StudentName":    "student_name",
		"StudentID":      "student_id",
		"TeamName":       "team_name",
		"Topic":          "topic",
		"TopicRU":        "topic_ru",
		"TopicKZ":        "topic_kz",
		"TopicEN":        "topic_en",
		"SupervisorName": "supervisor_name",
		"DepartmentName": "department_name",
		"GroupName":      "group_name",
	}

	for placeholder, dataKey := range projectDataFields {
		if val, ok := actx.ProjectData[dataKey]; ok {
			if strVal, ok := val.(string); ok && strVal != "" {
				replacements[placeholder] = strVal
			}
		}
	}

	// Дополнительные плейсхолдеры из конфига
	if customPlaceholders, ok := actx.Config["placeholders"].(map[string]interface{}); ok {
		for placeholder, dataKeyRaw := range customPlaceholders {
			if dataKey, ok := dataKeyRaw.(string); ok {
				if val, ok := actx.ProjectData[dataKey]; ok {
					if strVal, ok := val.(string); ok {
						replacements[placeholder] = strVal
					}
				}
			}
		}
	}

	// ===== Проверяем наличие шаблона =====
	templatePath := filepath.Join(templatesDir, templateName+".docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return &plugins.ActionResult{
			Success: true,
			Data: map[string]interface{}{
				"template_name":   templateName,
				"template_path":   templatePath,
				"output_filename": outputFilename,
				"replacements":    replacementsToMap(replacements),
				"status":          "template_not_found",
				"message":         fmt.Sprintf("Template '%s' not found at '%s'. Document generation skipped. Place the template and retry.", templateName, templatePath),
				"generated_at":    time.Now().UTC().Format(time.RFC3339),
			},
		}
	}

	// ===== Генерация DOCX =====
	doc, err := godocx.Open(templatePath)
	if err != nil {
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("failed to open template '%s': %w", templatePath, err),
		}
	}

	if err := doc.ReplaceAll(replacements); err != nil {
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("failed to replace placeholders: %w", err),
		}
	}

	// Создаём output директорию
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("failed to create output dir '%s': %w", outputDir, err),
		}
	}

	// Уникальное имя файла: project_id + timestamp
	uniqueName := fmt.Sprintf("p%d_%d_%s", actx.ProjectID, time.Now().Unix(), outputFilename)
	outputPath := filepath.Join(outputDir, uniqueName)

	if err := doc.WriteToFile(outputPath); err != nil {
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("failed to write document to '%s': %w", outputPath, err),
		}
	}

	// TODO (Phase 2): Upload to file_service via gRPC streaming and get file_id
	// fileID, err := uploadToFileService(ctx, outputPath, actx.ProjectID)

	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"template_name":   templateName,
			"output_filename": uniqueName,
			"output_path":     outputPath,
			"replacements":    replacementsToMap(replacements),
			"status":          "generated",
			"message":         "Document generated successfully",
			"generated_at":    time.Now().UTC().Format(time.RFC3339),
			// "file_id": fileID, // Phase 2
		},
	}
}

func (p *DocumentGeneratorPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

func replacementsToMap(pm godocx.PlaceholderMap) map[string]string {
	result := make(map[string]string, len(pm))
	for k, v := range pm {
		switch val := v.(type) {
		case string:
			result[k] = val
		default:
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func getCurrentAcademicYear() string {
	now := time.Now()
	year := now.Year()
	month := now.Month()
	if month >= time.September {
		return fmt.Sprintf("%d-%d", year, year+1)
	}
	return fmt.Sprintf("%d-%d", year-1, year)
}

// =============== Webhook Plugin ===============
type WebhookPlugin struct{}

func NewWebhookPlugin() *WebhookPlugin { return &WebhookPlugin{} }

func (p *WebhookPlugin) ID() string   { return "CALL_WEBHOOK" }
func (p *WebhookPlugin) Name() string { return "Вызов webhook" }
func (p *WebhookPlugin) Description() string {
	return "Отправляет HTTP запрос на внешний URL"
}
func (p *WebhookPlugin) Category() string   { return plugins.CategoryWebhook }
func (p *WebhookPlugin) IsReversible() bool { return false }

func (p *WebhookPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["url"],
		"properties": {
			"url": {"type": "string", "title": "URL"},
			"method": {"type": "string", "enum": ["GET", "POST", "PUT"], "default": "POST"},
			"headers": {"type": "object"},
			"body_template": {"type": "string"}
		}
	}`)
}

func (p *WebhookPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *WebhookPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"status": "stub",
			"reason": "webhook plugin not yet implemented",
		},
	}
}
func (p *WebhookPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }
