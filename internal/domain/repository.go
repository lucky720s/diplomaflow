// file: domain/repository.go
package domain

import "context"

// StudentRepository определяет методы для работы с хранилищем студентов
type StudentRepository interface {
	Create(ctx context.Context, student *Student) error
	GetByID(ctx context.Context, id string) (*Student, error)
	List(ctx context.Context) ([]*Student, error)
	Delete(ctx context.Context, id string) error
}

// И так далее для University и Department...
type DepartmentRepository interface {
	Create(ctx context.Context, department *Department) error
	GetByID(ctx context.Context, id string) (*Department, error)
	// TODO: List, Update, Delete
}

// DiplomaProjectRepository ... (без изменений)
type DiplomaProjectRepository interface {
	Create(ctx context.Context, project *DiplomaProject) error
	GetByID(ctx context.Context, id string) (*DiplomaProject, error)
	UpdateStatus(ctx context.Context, id, status string) error
}
