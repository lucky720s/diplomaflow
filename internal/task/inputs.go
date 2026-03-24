package task

import (
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// ==================== BOARD ====================

type CreateBoardInput struct {
	TeamID      int64
	ProjectID   int64
	Name        string
	Description string
	CreatedBy   int64
}

type UpdateBoardInput struct {
	BoardID     int64
	Name        string
	Description string
	UpdateMask  *fieldmaskpb.FieldMask
}

// ==================== COLUMN ====================

type CreateColumnInput struct {
	BoardID      int64
	Name         string
	Slug         string
	Description  string
	Color        string
	Icon         string
	WIPLimit     int32
	IsDoneColumn bool
}

type UpdateColumnInput struct {
	ColumnID    int64
	Name        string
	Description string
	Color       string
	Icon        string
	WIPLimit    int32
	UpdateMask  *fieldmaskpb.FieldMask
}

// ==================== TASK ====================

type CreateTaskInput struct {
	BoardID          int64
	ColumnID         int64
	Title            string
	Description      string
	Priority         string
	AssigneeID       int64
	DueDate          *time.Time
	EstimatedMinutes int32
	Labels           []string
	CreatedBy        int64
	WorkflowStepID   int64
}

type UpdateTaskInput struct {
	TaskID           int64
	Title            string
	Description      string
	Priority         *string
	DueDate          *time.Time
	EstimatedMinutes int32
	ActualMinutes    int32
	Labels           []string
	WorkflowStepID   int64
	UpdateMask       *fieldmaskpb.FieldMask
	UpdatedBy        int64
}

type MoveTaskInput struct {
	TaskID   int64
	ColumnID int64
	Position int
	MovedBy  int64
}

type BulkUpdateInput struct {
	TaskIDs        []int64
	AssigneeID     int64
	Priority       *string
	AddLabels      []string
	RemoveLabels   []string
	MoveToColumnID int64
	UpdatedBy      int64
}

// ==================== COMMENT ====================

type CreateCommentInput struct {
	TaskID         int64
	AuthorID       int64
	Content        string
	MentionUserIDs []int64
}

// ==================== ATTACHMENT ====================

type AddAttachmentInput struct {
	TaskID     int64
	FileID     string
	FileName   string
	FileType   string
	FileSize   int64
	UploadedBy int64
}

// ==================== FILTERS & COUNTS ====================
// TaskFilter, MyTasksFilter, TaskCounts определены в repository.go
// НЕ дублируем их здесь!
