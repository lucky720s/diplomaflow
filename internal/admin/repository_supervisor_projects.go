package admin

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Supervisor-facing project repository.
//
// Ownership model (NB: projects has no supervisor_id column): a supervisor owns
// a project when projects.team_id maps to admin_supervisor_assignments.supervisor_id.
// These queries always scope by that join — never by a client-supplied value.

// SupervisorProjectsFilter is the input for ListSupervisorProjects.
type SupervisorProjectsFilter struct {
	SupervisorID int64
	Status       string
	CurrentState string
	Search       string
	Limit        int
	Offset       int
	Sort         string // updated_at | created_at | title
	Order        string // asc | desc
}

// SupervisorProjectRow is one row of the supervisor projects list (flat, with stats).
type SupervisorProjectRow struct {
	ProjectID        int64  `gorm:"column:project_id"`
	Title            string `gorm:"column:title"`
	Description      string `gorm:"column:description"`
	Status           string `gorm:"column:status"`
	CurrentStateID   int64  `gorm:"column:current_state_id"`
	CurrentStateName string `gorm:"column:current_state_name"`
	CurrentDisplay   string `gorm:"column:current_display_name"`
	CurrentOrder     int32  `gorm:"column:current_order"`
	TotalSteps       int32  `gorm:"column:total_steps"`

	TeamID   int64  `gorm:"column:team_id"`
	TeamName string `gorm:"column:team_name"`

	SupervisorID int64 `gorm:"column:supervisor_id"`

	SubmissionsCount    int32      `gorm:"column:submissions_count"`
	PendingReviewsCount int32      `gorm:"column:pending_reviews_count"`
	FilesCount          int32      `gorm:"column:files_count"`
	GradesCount         int32      `gorm:"column:grades_count"`
	LastActivityAt      *time.Time `gorm:"column:last_activity_at"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// SupervisorProjectHead is the per-project header used to assemble the detail view.
type SupervisorProjectHead struct {
	ProjectID         int64      `gorm:"column:project_id"`
	Title             string     `gorm:"column:title"`
	Description       string     `gorm:"column:description"`
	Status            string     `gorm:"column:status"`
	CurrentStateID    int64      `gorm:"column:current_state_id"`
	CurrentStateName  string     `gorm:"column:current_state_name"`
	CurrentDisplay    string     `gorm:"column:current_display_name"`
	WorkflowID        int64      `gorm:"column:workflow_id"`
	WorkflowName      string     `gorm:"column:workflow_name"`
	TeamID            int64      `gorm:"column:team_id"`
	TeamName          string     `gorm:"column:team_name"`
	TeamStatus        string     `gorm:"column:team_status"`
	TeamCreatedAt     time.Time  `gorm:"column:team_created_at"`
	TeamUpdatedAt     time.Time  `gorm:"column:team_updated_at"`
	SupervisorID      int64      `gorm:"column:supervisor_id"`
	TopicRegisteredAt *time.Time `gorm:"column:topic_registered_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

// SupervisorProjectFileRow is one file aggregated from a project's submissions.
type SupervisorProjectFileRow struct {
	SubmissionID   string    `gorm:"column:submission_id"`
	StepID         int64     `gorm:"column:step_id"`
	StepName       string    `gorm:"column:step_name"`
	UploadedBy     int64     `gorm:"column:uploaded_by"`
	UploadedByName string    `gorm:"column:uploaded_by_name"`
	UploadedAt     time.Time `gorm:"column:uploaded_at"`
	Files          []byte    `gorm:"column:files"` // raw JSONB array of file attachments
}

// IsSupervisorOfProject reports whether the project exists at all, and whether
// it is owned by supervisorID. The two booleans let the caller distinguish a
// 404 (missing) from a 403 (exists but not owned).
func (r *repository) IsSupervisorOfProject(ctx context.Context, supervisorID, projectID int64) (exists bool, owned bool, err error) {
	var row struct {
		Exists bool `gorm:"column:exists"`
		Owned  bool `gorm:"column:owned"`
	}
	q := `
		SELECT
			EXISTS(SELECT 1 FROM projects WHERE id = ?) AS exists,
			EXISTS(
				SELECT 1
				FROM projects p
				JOIN admin_supervisor_assignments asa ON asa.team_id = p.team_id
				WHERE p.id = ? AND asa.supervisor_id = ?
			) AS owned`
	if e := r.db.WithContext(ctx).Raw(q, projectID, projectID, supervisorID).Scan(&row).Error; e != nil {
		return false, false, e
	}
	return row.Exists, row.Owned, nil
}

// ListSupervisorProjects returns the calling supervisor's projects with stats.
func (r *repository) ListSupervisorProjects(ctx context.Context, f SupervisorProjectsFilter) ([]*SupervisorProjectRow, int64, error) {
	base := `
		FROM projects p
		JOIN admin_supervisor_assignments asa ON asa.team_id = p.team_id
		LEFT JOIN teams t ON t.id = p.team_id
		LEFT JOIN states cs ON cs.id = p.current_state_id
		WHERE asa.supervisor_id = ?`
	args := []interface{}{f.SupervisorID}

	if f.Status != "" {
		base += " AND p.status = ?"
		args = append(args, f.Status)
	} else {
		// Hide archived/deleted projects unless explicitly requested.
		base += " AND p.status <> 'archived'"
	}
	if f.CurrentState != "" {
		base += " AND p.current_state_name = ?"
		args = append(args, f.CurrentState)
	}
	if f.Search != "" {
		base += " AND (p.title ILIKE ? OR p.description ILIKE ?)"
		like := "%" + f.Search + "%"
		args = append(args, like, like)
	}

	var total int64
	if err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) "+base, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*SupervisorProjectRow{}, 0, nil
	}

	selectSQL := `
		SELECT
			p.id AS project_id,
			p.title AS title,
			COALESCE(p.description, '') AS description,
			p.status AS status,
			COALESCE(p.current_state_id, 0) AS current_state_id,
			COALESCE(p.current_state_name, '') AS current_state_name,
			COALESCE(cs.display_name, p.current_state_name, '') AS current_display_name,
			COALESCE(cs.order_index, 0) AS current_order,
			COALESCE((SELECT COUNT(*) FROM states s WHERE s.workflow_id = p.workflow_id AND s.deleted_at IS NULL), 0) AS total_steps,
			COALESCE(p.team_id, 0) AS team_id,
			COALESCE(t.name, '') AS team_name,
			asa.supervisor_id AS supervisor_id,
			COALESCE((SELECT COUNT(*) FROM admin_submissions sub WHERE sub.project_id = p.id AND sub.deleted_at IS NULL), 0) AS submissions_count,
			COALESCE((SELECT COUNT(*) FROM admin_submissions sub WHERE sub.project_id = p.id AND sub.deleted_at IS NULL AND sub.status = 'pending'), 0) AS pending_reviews_count,
			COALESCE((SELECT SUM(jsonb_array_length(sub.files)) FROM admin_submissions sub WHERE sub.project_id = p.id AND sub.deleted_at IS NULL AND jsonb_typeof(sub.files) = 'array'), 0) AS files_count,
			COALESCE((SELECT COUNT(*) FROM admin_grades g WHERE g.project_id = p.id AND g.deleted_at IS NULL), 0) AS grades_count,
			GREATEST(p.updated_at, COALESCE((SELECT MAX(sub.updated_at) FROM admin_submissions sub WHERE sub.project_id = p.id AND sub.deleted_at IS NULL), p.updated_at)) AS last_activity_at,
			p.created_at AS created_at,
			p.updated_at AS updated_at`

	order := buildSupervisorProjectsOrder(f.Sort, f.Order)
	full := selectSQL + " " + base + " " + order
	if f.Limit > 0 {
		full += fmt.Sprintf(" LIMIT %d", f.Limit)
	}
	if f.Offset > 0 {
		full += fmt.Sprintf(" OFFSET %d", f.Offset)
	}

	var rows []*SupervisorProjectRow
	if err := r.db.WithContext(ctx).Raw(full, args...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// buildSupervisorProjectsOrder whitelists sort/order to avoid SQL injection.
func buildSupervisorProjectsOrder(sort, order string) string {
	col := "p.updated_at"
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "created_at":
		col = "p.created_at"
	case "title":
		col = "p.title"
	case "updated_at", "":
		col = "p.updated_at"
	}
	dir := "DESC"
	if strings.ToLower(strings.TrimSpace(order)) == "asc" {
		dir = "ASC"
	}
	return "ORDER BY " + col + " " + dir
}

// GetSupervisorProjectHead loads the per-project header for the detail view.
func (r *repository) GetSupervisorProjectHead(ctx context.Context, projectID int64) (*SupervisorProjectHead, error) {
	var head SupervisorProjectHead
	q := `
		SELECT
			p.id AS project_id,
			p.title AS title,
			COALESCE(p.description, '') AS description,
			p.status AS status,
			COALESCE(p.current_state_id, 0) AS current_state_id,
			COALESCE(p.current_state_name, '') AS current_state_name,
			COALESCE(cs.display_name, p.current_state_name, '') AS current_display_name,
			COALESCE(p.workflow_id, 0) AS workflow_id,
			COALESCE(p.workflow_name, '') AS workflow_name,
			COALESCE(p.team_id, 0) AS team_id,
			COALESCE(t.name, '') AS team_name,
			COALESCE(p.status, 'active') AS team_status,
			COALESCE(t.created_at, p.created_at) AS team_created_at,
			COALESCE(t.updated_at, p.updated_at) AS team_updated_at,
			COALESCE(asa.supervisor_id, 0) AS supervisor_id,
			p.topic_registered_at AS topic_registered_at,
			p.created_at AS created_at,
			p.updated_at AS updated_at
		FROM projects p
		LEFT JOIN teams t ON t.id = p.team_id
		LEFT JOIN states cs ON cs.id = p.current_state_id
		LEFT JOIN admin_supervisor_assignments asa ON asa.team_id = p.team_id
		WHERE p.id = ?`
	if err := r.db.WithContext(ctx).Raw(q, projectID).Scan(&head).Error; err != nil {
		return nil, err
	}
	return &head, nil
}

// ListSupervisorProjectFiles returns submissions (with raw files JSONB) for a
// project, newest first, so the service can flatten them into file entries.
func (r *repository) ListSupervisorProjectFiles(ctx context.Context, projectID int64) ([]*SupervisorProjectFileRow, error) {
	q := `
		SELECT
			s.id AS submission_id,
			COALESCE(s.state_id, 0) AS step_id,
			COALESCE(st.name, '') AS step_name,
			s.submitted_by AS uploaded_by,
			COALESCE(CONCAT(u.first_name, ' ', u.last_name), '') AS uploaded_by_name,
			s.created_at AS uploaded_at,
			s.files AS files
		FROM admin_submissions s
		LEFT JOIN states st ON st.id = s.state_id
		LEFT JOIN users u ON u.id = s.submitted_by
		WHERE s.project_id = ? AND s.deleted_at IS NULL
		  AND jsonb_typeof(s.files) = 'array' AND jsonb_array_length(s.files) > 0
		ORDER BY s.created_at DESC`
	var rows []*SupervisorProjectFileRow
	if err := r.db.WithContext(ctx).Raw(q, projectID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
