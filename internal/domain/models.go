package domain

// Student представляет студента с кафедрой
type Student struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	DepartmentID string `json:"department_id"`
}

// Department представляет кафедру
type Department struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	UniversityID string `json:"university_id"`
}

// University представляет университет
type University struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DiplomaProject представляет дипломный проект
type DiplomaProject struct {
	ID          string `json:"id"`
	StudentID   string `json:"student_id"`
	Topic       string `json:"topic"`
	Status      string `json:"status"` // draft, approved, completed
	Grade       int    `json:"grade"`
	Reviewer    string `json:"reviewer"`
	ReviewerID  string `json:"reviewer_id"`
	Advisor     string `json:"advisor"`
	AdvisorID   string `json:"advisor_id"`
	DefenseDate string `json:"defense_date"`
}

// --- ДОБАВЛЕННЫЕ МОДЕЛИ ДЛЯ АВТОРИЗАЦИИ ---

// Role определяет роль пользователя в системе
type Role string

const (
	RoleStudent      Role = "student"
	RoleSupervisor   Role = "supervisor"
	RoleDeptReviewer Role = "dept_reviewer"
	RoleUniReviewer  Role = "uni_reviewer"
	RoleDeptAdmin    Role = "dept_admin"
	RoleSysAdmin     Role = "sys_admin"
)

// User представляет пользователя в контексте запроса, извлеченного из токена
type User struct {
	ID   string
	Role Role
}
