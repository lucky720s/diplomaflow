package domain

import "github.com/google/uuid"

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
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         Role      `json:"role"`
}

type University struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type Department struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	UniversityID uuid.UUID `json:"university_id"`
}

type StudentProfile struct {
	User
	Department Department `json:"department"`
}

type StaffProfile struct {
	User
	Department Department `json:"department"`
}

type DiplomaProject struct {
	ID           uuid.UUID `json:"id"`
	Topic        string    `json:"topic"`
	Status       string    `json:"status"`
	StudentID    uuid.UUID `json:"student_id"`
	SupervisorID uuid.UUID `json:"supervisor_id"`
}
