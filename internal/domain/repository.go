// file: domain/repository.go
package domain

import "context"

// StudentRepository определяет методы для работы с хранилищем студентов
type StudentRepository interface {
	// Create сохраняет нового студента в хранилище
	Create(ctx context.Context, student *Student) error
	// GetByID находит студента по его ID
	GetByID(ctx context.Context, id string) (*Student, error)
	// ... другие методы: Update, Delete, ListByDepartmentID и т.д.
}

// DiplomaProjectRepository определяет методы для работы с дипломными проектами
type DiplomaProjectRepository interface {
	Create(ctx context.Context, project *DiplomaProject) error
	GetByID(ctx context.Context, id string) (*DiplomaProject, error)
	UpdateStatus(ctx context.Context, id, status string) error
	// ...
}

// И так далее для University и Department...
