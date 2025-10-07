package domain

type Role string

const (
	RoleStudent      Role = "student"
	RoleSupervisor   Role = "supervisor"
	RoleDeptReviewer Role = "dept_reviewer"
	RoleUniReviewer  Role = "uni_reviewer"
	RoleDeptAdmin    Role = "dept_admin"
	RoleSysAdmin     Role = "sys_admin"
)

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	FullName     string `json:"full_name"`
}

type UserRole struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`
	Role   Role  `json:"role"`
}

type University struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Faculty struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	UniversityID int64  `json:"university_id"`
}

type Department struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	FacultyID int64  `json:"faculty_id"`
}

type StudentProfile struct {
	User
	Department Department `json:"department"`
}

type StaffProfile struct {
	User
	Department Department `json:"department"`
}

type Team struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	LeaderID int64  `json:"leader_id"`
}

type TeamMember struct {
	ID     int64 `json:"id"`
	TeamID int64 `json:"team_id"`
	UserID int64 `json:"user_id"`
}

type DiplomaProject struct {
	ID           int64  `json:"id"`
	Topic        string `json:"topic"`
	SupervisorID int64  `json:"supervisor_id"`
	TeamID       int64  `json:"team_id"`
}

type ProgressStage string

const (
	StageTeamSelection    ProgressStage = "Выбор команды"
	StageSupervisorChoice ProgressStage = "Выбор руководителя"
	StageTopicSelection   ProgressStage = "Выбор темы"
	StageEnglish          ProgressStage = "Английский язык"
	StageEconomics        ProgressStage = "Экономическая часть"
	StageAntiplagiarism   ProgressStage = "Антиплагиат"
	StageNormControl      ProgressStage = "Нормаконтроль"
	StageDefensePending   ProgressStage = "Ожидает защиты"
	StageDefended         ProgressStage = "Защищено"
)

type ProjectProgress struct {
	ID             int64         `json:"id"`
	DiplomaProject int64         `json:"diploma_project_id"`
	CurrentStage   ProgressStage `json:"current_stage"`
}
