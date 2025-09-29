package domain

// Student представляет сущность студента
type Student struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	DepartmentID string `json:"department_id"` // <-- УБЕДИТЕСЬ, ЧТО ЭТО ПОЛЕ НА МЕСТЕ
}

// Department представляет сущность кафедры
type Department struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	UniversityID string `json:"university_id"`
}

// University представляет сущность университета
type University struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DiplomaProject представляет сущность дипломного проекта
type DiplomaProject struct {
	ID          string `json:"id"`
	StudentID   string `json:"student_id"`
	Topic       string `json:"topic"`
	Status      string `json:"status"` // например, "draft", "approved", "completed"
	Grade       int    `json:"grade"`
	Reviewer    string `json:"reviewer"`
	ReviewerID  string `json:"reviewer_id"`
	Advisor     string `json:"advisor"`
	AdvisorID   string `json:"advisor_id"`
	DefenseDate string `json:"defense_date"`
}

// НОВАЯ СТРУКТУРА, КОТОРУЮ МЫ ЗАБЫЛИ ДОБАВИТЬ
type StudentWithDepartment struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	// Встраиваем информацию о кафедре
	Department struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"department"`
}
