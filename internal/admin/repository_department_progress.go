package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ==========================================================================
// Department Progress repository (teacher-facing, read-only aggregates).
//
// All reads are department-scoped. The list query is a single paginated CTE
// (no N+1); per-team relations (members, supervisor) are batch-loaded by id.
// ==========================================================================

// admissionCaseSQL is the SQL expression that normalizes an admission decision.
// IMPORTANT: this MUST stay in sync with normalizeAdmission() in
// service_department_progress.go (used for the team-details view).
const admissionCaseSQL = `
	CASE
		WHEN %[1]s.pre_defense_result = 'passed'      THEN 'admitted'
		WHEN %[1]s.pre_defense_result = 'failed'      THEN 'not_admitted'
		WHEN %[1]s.pre_defense_result = 'conditional' THEN 'revision_required'
		WHEN %[1]s.adm_grade IS NOT NULL AND %[1]s.adm_grade >= %[1]s.adm_passing THEN 'admitted'
		WHEN %[1]s.adm_grade IS NOT NULL AND %[1]s.adm_grade <  %[1]s.adm_passing THEN 'not_admitted'
		ELSE 'not_decided'
	END`

// dpLateralJoins is the shared set of LATERAL subqueries that resolve the
// "latest row per project" used by both the list and the summary admission tally.
const dpLateralJoins = `
	LEFT JOIN states cs ON cs.id = p.current_state_id
	LEFT JOIN admin_supervisor_assignments sa ON sa.team_id = t.id
	LEFT JOIN LATERAL (
		SELECT status FROM admin_topic_registrations
		WHERE project_id = p.id AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1
	) tr ON TRUE
	LEFT JOIN LATERAL (
		SELECT status FROM norm_control_checks WHERE project_id = p.id ORDER BY created_at DESC LIMIT 1
	) nc ON TRUE
	LEFT JOIN LATERAL (
		SELECT status, plagiarism_score FROM antiplag_checks WHERE project_id = p.id ORDER BY created_at DESC LIMIT 1
	) ap ON TRUE
	LEFT JOIN LATERAL (
		SELECT status, result, grade FROM admin_pre_defense_submissions WHERE project_id = p.id ORDER BY created_at DESC LIMIT 1
	) pd ON TRUE
	LEFT JOIN LATERAL (
		SELECT g.grade FROM admin_grades g JOIN states s2 ON s2.id = g.state_id
		WHERE g.project_id = p.id AND g.deleted_at IS NULL AND s2.is_final = TRUE
		ORDER BY g.created_at DESC LIMIT 1
	) fg ON TRUE
	LEFT JOIN LATERAL (
		SELECT g.grade, COALESCE((s2.config->'review_config'->>'passing_score')::int, 0) AS passing_score
		FROM admin_grades g JOIN states s2 ON s2.id = g.state_id
		WHERE g.project_id = p.id AND g.deleted_at IS NULL
		  AND (s2.config->'review_config'->>'grade_type') = 'admission'
		ORDER BY g.created_at DESC LIMIT 1
	) adm ON TRUE
	LEFT JOIN LATERAL (
		SELECT MAX(created_at) AS last_activity_at FROM state_histories WHERE project_id = p.id
	) la ON TRUE`

// sort_by -> safe column (whitelist; never interpolate user input directly).
var dpSortColumns = map[string]string{
	"team_name":           "team_name",
	"project_title":       "project_title",
	"current_stage":       "current_state_name",
	"progress_percentage": "progress_percentage",
	"last_activity_at":    "last_activity_at",
	"final_grade":         "final_grade",
	"admission_status":    "admission_status",
}

func (r *repository) ListDepartmentProgressTeams(ctx context.Context, f DepartmentProgressTeamFilter) ([]*dpTeamRow, int64, error) {
	cte := `
	WITH base AS (
		SELECT
			t.id AS team_id, t.name AS team_name, t.created_at AS created_at, t.updated_at AS updated_at,
			COALESCE(p.id, 0) AS project_id, COALESCE(p.title, '') AS project_title,
			COALESCE(p.current_state_name, '') AS current_state_name,
			cs.order_index AS current_order,
			COALESCE(sa.supervisor_id, 0) AS supervisor_id,
			COALESCE(tr.status, '') AS topic_status,
			COALESCE(nc.status, '') AS norm_control_status,
			COALESCE(ap.status, '') AS antiplagiat_status,
			ap.plagiarism_score AS antiplagiat_percent,
			COALESCE(pd.status, '') AS pre_defense_status,
			pd.result AS pre_defense_result,
			pd.grade AS pre_defense_grade,
			fg.grade AS final_grade,
			adm.grade AS adm_grade,
			COALESCE(adm.passing_score, 0) AS adm_passing,
			la.last_activity_at AS last_activity_at,
			(SELECT COUNT(*) FROM states ss WHERE ss.workflow_id = p.workflow_id AND ss.deleted_at IS NULL) AS total_steps
		FROM teams t
		LEFT JOIN projects p ON p.team_id = t.id` + dpLateralJoins + `
		WHERE t.department_id = ? AND t.deleted_at IS NULL
	), computed AS (
		SELECT b.*,
			` + fmt.Sprintf(admissionCaseSQL, "b") + ` AS admission_status,
			CASE WHEN b.total_steps > 0 AND b.current_order IS NOT NULL
				THEN LEAST(100, ROUND(((b.current_order + 1.0) / b.total_steps) * 100))::float8
				ELSE 0 END AS progress_percentage
		FROM base b
	)`

	// Dynamic filters over the computed projection.
	var conds []string
	args := []interface{}{f.DepartmentID}

	if s := strings.TrimSpace(f.Search); s != "" {
		conds = append(conds, "(team_name ILIKE ? OR project_title ILIKE ?)")
		like := "%" + s + "%"
		args = append(args, like, like)
	}
	if f.SupervisorID > 0 {
		conds = append(conds, "supervisor_id = ?")
		args = append(args, f.SupervisorID)
	}
	if v := strings.TrimSpace(f.CurrentStage); v != "" {
		conds = append(conds, "current_state_name = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.TopicStatus); v != "" {
		conds = append(conds, "topic_status = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.NormControlStatus); v != "" {
		conds = append(conds, "norm_control_status = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.AntiplagiatStatus); v != "" {
		conds = append(conds, "antiplagiat_status = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.PreDefenseStatus); v != "" {
		conds = append(conds, "pre_defense_status = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.AdmissionStatus); v != "" {
		conds = append(conds, "admission_status = ?")
		args = append(args, v)
	}
	if f.MinGrade != nil {
		conds = append(conds, "final_grade >= ?")
		args = append(args, *f.MinGrade)
	}
	if f.MaxGrade != nil {
		conds = append(conds, "final_grade <= ?")
		args = append(args, *f.MaxGrade)
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	// Sorting (whitelisted column + direction).
	col, ok := dpSortColumns[strings.TrimSpace(f.SortBy)]
	if !ok {
		col = "last_activity_at"
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(f.SortOrder), "asc") {
		dir = "ASC"
	}
	order := fmt.Sprintf(" ORDER BY %s %s NULLS LAST, team_id ASC", col, dir)

	// Count (same filters, no pagination).
	var total int64
	countSQL := cte + " SELECT COUNT(*) FROM computed" + where
	if err := r.db.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Page.
	offset := int((f.Page - 1) * f.PageSize)
	listArgs := append(append([]interface{}{}, args...), f.PageSize, offset)
	listSQL := cte + " SELECT * FROM computed" + where + order + " LIMIT ? OFFSET ?"

	var rows []*dpTeamRow
	if err := r.db.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *repository) GetDPTeamMembers(ctx context.Context, teamIDs []int64) (map[int64][]*DPMember, error) {
	out := make(map[int64][]*DPMember)
	if len(teamIDs) == 0 {
		return out, nil
	}
	type row struct {
		TeamID   int64  `gorm:"column:team_id"`
		UserID   int64  `gorm:"column:user_id"`
		FullName string `gorm:"column:full_name"`
		Email    string `gorm:"column:email"`
		Role     string `gorm:"column:role"`
	}
	q := `
		SELECT tm.team_id, tm.user_id,
		       COALESCE(CONCAT(u.first_name, ' ', u.last_name), '') AS full_name,
		       COALESCE(u.email, '') AS email, tm.role
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id IN (?)
		ORDER BY tm.team_id, (tm.role = 'leader') DESC, tm.created_at ASC`
	var rows []row
	if err := r.db.WithContext(ctx).Raw(q, teamIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, m := range rows {
		out[m.TeamID] = append(out[m.TeamID], &DPMember{
			UserID: m.UserID, FullName: m.FullName, Email: m.Email, Role: m.Role,
		})
	}
	return out, nil
}

func (r *repository) GetDPSupervisors(ctx context.Context, supervisorIDs []int64) (map[int64]*DPUser, error) {
	out := make(map[int64]*DPUser)
	if len(supervisorIDs) == 0 {
		return out, nil
	}
	type row struct {
		ID       int64  `gorm:"column:id"`
		FullName string `gorm:"column:full_name"`
		Email    string `gorm:"column:email"`
	}
	q := `
	SELECT u.id,
	       COALESCE(CONCAT(u.first_name, ' ', u.last_name), '') AS full_name,
	       COALESCE(u.email, '') AS email
	FROM users u
	WHERE u.id IN (?)`
	var rows []row
	if err := r.db.WithContext(ctx).Raw(q, supervisorIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, u := range rows {
		out[u.ID] = &DPUser{ID: u.ID, FullName: u.FullName, Email: u.Email, Position: "Teacher"}
	}
	return out, nil
}

// GetDPTeamHead loads the team header guarded by department (cross-department guard).
// Returns gorm.ErrRecordNotFound when the team is absent or in another department.
func (r *repository) GetDPTeamHead(ctx context.Context, teamID, departmentID int64) (*DPTeamInfo, error) {
	var info DPTeamInfo
	q := `
		SELECT t.id AS team_id, t.name AS team_name,
		       COALESCE(p.status, 'active') AS status,
		       t.created_at AS created_at, t.updated_at AS updated_at,
		       COALESCE(p.id, 0) AS project_id, COALESCE(p.title, '') AS project_title,
		       COALESCE(p.description, '') AS project_desc,
		       COALESCE(p.current_state_id, 0) AS current_state_id,
		       COALESCE(p.current_state_name, '') AS current_state_name,
		       COALESCE(p.workflow_id, 0) AS workflow_id,
		       COALESCE(sa.supervisor_id, 0) AS supervisor_id
		FROM teams t
		LEFT JOIN projects p ON p.team_id = t.id
		LEFT JOIN admin_supervisor_assignments sa ON sa.team_id = t.id
		WHERE t.id = ? AND t.department_id = ? AND t.deleted_at IS NULL`
	if err := r.db.WithContext(ctx).Raw(q, teamID, departmentID).Scan(&info).Error; err != nil {
		return nil, err
	}
	if info.TeamID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &info, nil
}

func (r *repository) GetDPWorkflowSteps(ctx context.Context, projectID, workflowID int64) ([]*DPStep, error) {
	if workflowID == 0 {
		return []*DPStep{}, nil
	}
	q := `
		SELECT s.id AS step_id, s.name AS step_name, COALESCE(s.display_name, '') AS display_name,
		       s.order_index AS order_index, s.type AS step_type,
		       COALESCE(s.config->'review_config'->>'grade_type', '') AS grade_type,
		       COALESCE((s.config->'review_config'->>'passing_score')::int, 0) AS passing_score,
		       COALESCE(sub.status, '') AS submission_status,
		       g.grade AS grade,
		       COALESCE(CONCAT(ru.first_name, ' ', ru.last_name), '') AS reviewer_name,
		       sub.reviewed_at AS reviewed_at,
		       COALESCE(sub.review_comment, '') AS comment
		FROM states s
		LEFT JOIN LATERAL (
			SELECT status, reviewer_id, reviewed_at, review_comment
			FROM admin_submissions WHERE project_id = ? AND state_id = s.id ORDER BY created_at DESC LIMIT 1
		) sub ON TRUE
		LEFT JOIN admin_grades g ON g.project_id = ? AND g.state_id = s.id AND g.deleted_at IS NULL
		LEFT JOIN users ru ON ru.id = sub.reviewer_id
		WHERE s.workflow_id = ? AND s.deleted_at IS NULL
		ORDER BY s.order_index`
	var steps []*DPStep
	if err := r.db.WithContext(ctx).Raw(q, projectID, projectID, workflowID).Scan(&steps).Error; err != nil {
		return nil, err
	}
	return steps, nil
}

func (r *repository) GetDPTopicRegistration(ctx context.Context, projectID int64) (*DPTopicRegistration, error) {
	if projectID == 0 {
		return nil, nil
	}
	q := `
		SELECT COALESCE(tr.status, '') AS status,
		       COALESCE(tr.proposed_topic_kz, '') AS proposed_topic_kz,
		       COALESCE(tr.proposed_topic_ru, '') AS proposed_topic_ru,
		       COALESCE(tr.proposed_topic_en, '') AS proposed_topic_en,
		       COALESCE(CONCAT(ru.first_name, ' ', ru.last_name), '') AS reviewer_name,
		       tr.reviewed_at AS reviewed_at,
		       COALESCE(tr.comment, '') AS comment,
		       COALESCE(tr.rejection_reason, '') AS rejection_reason
		FROM admin_topic_registrations tr
		LEFT JOIN users ru ON ru.id = tr.reviewer_id
		WHERE tr.project_id = ? AND tr.deleted_at IS NULL
		ORDER BY tr.created_at DESC LIMIT 1`
	rows := []*DPTopicRegistration{}
	if err := r.db.WithContext(ctx).Raw(q, projectID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *repository) GetDPNormControl(ctx context.Context, projectID int64) (*DPNormControl, error) {
	if projectID == 0 {
		return nil, nil
	}
	q := `
		SELECT COALESCE(nc.status, '') AS status,
		       COALESCE(nc.submission_id, '') AS submission_id,
		       COALESCE(CONCAT(cu.first_name, ' ', cu.last_name), '') AS reviewer_name,
		       nc.reviewed_at AS reviewed_at,
		       (SELECT COUNT(*) FROM norm_control_issues i WHERE i.submission_id = nc.submission_id) AS issues_count,
		       (SELECT COUNT(*) FROM norm_control_issues i WHERE i.submission_id = nc.submission_id AND i.severity = 'critical') AS critical_issues_count
		FROM norm_control_checks nc
		LEFT JOIN users cu ON cu.id = nc.checker_id
		WHERE nc.project_id = ?
		ORDER BY nc.created_at DESC LIMIT 1`
	rows := []*DPNormControl{}
	if err := r.db.WithContext(ctx).Raw(q, projectID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *repository) GetDPAntiplagiat(ctx context.Context, projectID int64) (*DPAntiplagiat, error) {
	if projectID == 0 {
		return nil, nil
	}
	q := `
		SELECT COALESCE(ap.status, '') AS status,
		       ap.plagiarism_score AS similarity_percent,
		       ap.ai_score AS ai_percent,
		       ap.reviewed_at AS checked_at,
		       COALESCE(CONCAT(cu.first_name, ' ', cu.last_name), '') AS reviewer_name
		FROM antiplag_checks ap
		LEFT JOIN users cu ON cu.id = ap.checker_id
		WHERE ap.project_id = ?
		ORDER BY ap.created_at DESC LIMIT 1`
	rows := []*DPAntiplagiat{}
	if err := r.db.WithContext(ctx).Raw(q, projectID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *repository) GetDPPreDefenses(ctx context.Context, projectID int64) ([]*DPPreDefense, error) {
	if projectID == 0 {
		return []*DPPreDefense{}, nil
	}
	q := `
		SELECT id, status, scheduled_date, COALESCE(location, '') AS location,
		       grade, COALESCE(result, '') AS result, COALESCE(result_comment, '') AS comment
		FROM admin_pre_defense_submissions
		WHERE project_id = ?
		ORDER BY created_at DESC`
	var rows []*DPPreDefense
	if err := r.db.WithContext(ctx).Raw(q, projectID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return rows, nil
	}

	ids := make([]string, 0, len(rows))
	idx := make(map[string]*DPPreDefense, len(rows))
	for _, pd := range rows {
		ids = append(ids, pd.ID)
		idx[pd.ID] = pd
	}

	commQ := `
		SELECT c.submission_id, c.id, c.user_id,
		       COALESCE(CONCAT(u.first_name, ' ', u.last_name), '') AS full_name,
		       COALESCE(u.email, '') AS email, c.role, c.is_present, COALESCE(c.comment, '') AS comment
		FROM admin_pre_defense_commission c
		JOIN users u ON u.id = c.user_id
		WHERE c.submission_id IN (?)
		ORDER BY c.id`
	type commRow struct {
		SubmissionID string `gorm:"column:submission_id"`
		ID           int64  `gorm:"column:id"`
		UserID       int64  `gorm:"column:user_id"`
		FullName     string `gorm:"column:full_name"`
		Email        string `gorm:"column:email"`
		Role         string `gorm:"column:role"`
		IsPresent    bool   `gorm:"column:is_present"`
		Comment      string `gorm:"column:comment"`
	}
	var commRows []commRow
	if err := r.db.WithContext(ctx).Raw(commQ, ids).Scan(&commRows).Error; err != nil {
		return nil, err
	}
	for _, c := range commRows {
		if pd, ok := idx[c.SubmissionID]; ok {
			pd.Commission = append(pd.Commission, &PreDefenseCommissionMemberData{
				ID: c.ID, UserID: c.UserID, FullName: c.FullName, Email: c.Email,
				Role: c.Role, IsPresent: c.IsPresent, Comment: c.Comment,
			})
		}
	}
	return rows, nil
}

func (r *repository) GetDPProjectDepartmentID(ctx context.Context, projectID int64) (int64, error) {
	var deptID int64
	err := r.db.WithContext(ctx).
		Table("projects").
		Select("COALESCE(department_id, 0)").
		Where("id = ?", projectID).
		Scan(&deptID).Error
	return deptID, err
}

// GetDPUnifiedHistory merges every per-source history table into one timeline.
// Sources without rows simply contribute nothing (no synthetic data).
func (r *repository) GetDPUnifiedHistory(ctx context.Context, projectID, teamID int64) ([]*UnifiedHistoryItem, error) {
	q := `
		SELECT * FROM (
			SELECT sh.id::text AS id, 'project' AS source, COALESCE(sh.event_name, '') AS action,
			       COALESCE(sh.changed_by, 0) AS actor_id,
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), '') AS actor_name,
			       COALESCE(sh.from_state_name, '') AS old_value, COALESCE(sh.to_state_name, '') AS new_value,
			       COALESCE(sh.comment, '') AS comment, sh.created_at AS created_at
			FROM state_histories sh LEFT JOIN users u ON u.id = sh.changed_by
			WHERE sh.project_id = ?

			UNION ALL
			SELECT gh.id::text, 'grade', 'grade_changed', COALESCE(gh.changed_by, 0),
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), ''),
			       COALESCE(gh.old_grade::text, ''), gh.new_grade::text,
			       COALESCE(gh.reason, ''), gh.created_at
			FROM admin_grade_history gh LEFT JOIN users u ON u.id = gh.changed_by
			WHERE gh.project_id = ?

			UNION ALL
			SELECT trr.id::text, 'topic', trr.action, COALESCE(trr.reviewer_id, 0),
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), ''), '', '',
			       COALESCE(trr.comment, ''), trr.created_at
			FROM admin_topic_registration_reviews trr
			JOIN admin_topic_registrations tr ON tr.id = trr.registration_id
			LEFT JOIN users u ON u.id = trr.reviewer_id
			WHERE tr.project_id = ?

			UNION ALL
			SELECT srh.id::text, 'supervisor_request', srh.action, COALESCE(srh.actor_id, 0),
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), ''), '', '',
			       COALESCE(srh.comment, ''), srh.created_at
			FROM admin_supervisor_request_history srh
			JOIN admin_supervisor_requests sr ON sr.id = srh.request_id
			LEFT JOIN users u ON u.id = srh.actor_id
			WHERE sr.team_id = ?

			UNION ALL
			SELECT nch.id::text, 'norm_control', nch.action, COALESCE(nch.actor_id, 0),
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), ''), '', '',
			       COALESCE(nch.comment, ''), nch.created_at
			FROM norm_control_history nch LEFT JOIN users u ON u.id = nch.actor_id
			WHERE nch.project_id = ?

			UNION ALL
			SELECT aph.id::text, 'antiplagiat', aph.action, COALESCE(aph.actor_id, 0),
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), ''), '', '',
			       COALESCE(aph.comment, ''), aph.created_at
			FROM antiplag_history aph LEFT JOIN users u ON u.id = aph.actor_id
			WHERE aph.project_id = ?

			UNION ALL
			SELECT pdh.id::text, 'pre_defense', pdh.action, COALESCE(pdh.actor_id, 0),
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), ''),
			       COALESCE(pdh.old_value, ''), COALESCE(pdh.new_value, ''),
			       COALESCE(pdh.comment, ''), pdh.created_at
			FROM admin_pre_defense_history pdh
			JOIN admin_pre_defense_submissions pds ON pds.id = pdh.submission_id
			LEFT JOIN users u ON u.id = pdh.actor_id
			WHERE pds.project_id = ?

			UNION ALL
			SELECT sr.id::text, 'submission', sr.action, COALESCE(sr.reviewer_id, 0),
			       COALESCE(CONCAT(u.first_name, ' ', u.last_name), ''), '',
			       COALESCE(sr.grade::text, ''), COALESCE(sr.comment, ''), sr.created_at
			FROM admin_submission_reviews sr
			JOIN admin_submissions s ON s.id = sr.submission_id
			LEFT JOIN users u ON u.id = sr.reviewer_id
			WHERE s.project_id = ?
		) h
		ORDER BY h.created_at DESC`

	var items []*UnifiedHistoryItem
	args := []interface{}{projectID, projectID, projectID, teamID, projectID, projectID, projectID, projectID}
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetDepartmentProgressSummaryStats reuses the dashboard counts and adds the
// progress-specific ones (norm/antiplagiat pending, admission tally, avg grade).
func (r *repository) GetDepartmentProgressSummaryStats(ctx context.Context, departmentID int64) (*DepartmentProgressStatsData, error) {
	dash, err := r.GetDashboardStats(ctx, departmentID)
	if err != nil {
		return nil, err
	}

	out := &DepartmentProgressStatsData{
		TotalStudents:             dash.TotalStudents,
		TotalTeams:                dash.TotalTeams,
		TotalProjects:             dash.TotalProjects,
		CompletedProjects:         dash.CompletedProjects,
		PendingTopicRegistrations: dash.PendingTopicRegistrations,
		PendingSupervisorRequests: dash.PendingSupervisorRequests,
		PendingPreDefenses:        dash.PendingPreDefenses,
		ScheduledPreDefenses:      dash.ScheduledPreDefenses,
	}

	var pendingNorm int64
	if err := r.db.WithContext(ctx).
		Table("norm_control_checks nc").
		Joins("JOIN projects p ON p.id = nc.project_id").
		Where("p.department_id = ? AND nc.status IN ?", departmentID, []string{"submitted", "in_review"}).
		Count(&pendingNorm).Error; err != nil {
		return nil, err
	}
	out.PendingNormControl = int32(pendingNorm)

	var pendingAntiplag int64
	if err := r.db.WithContext(ctx).
		Table("antiplag_checks ap").
		Joins("JOIN projects p ON p.id = ap.project_id").
		Where("p.department_id = ? AND ap.status IN ?", departmentID, []string{"submitted", "in_review"}).
		Count(&pendingAntiplag).Error; err != nil {
		return nil, err
	}
	out.PendingAntiplagiat = int32(pendingAntiplag)

	// Admission tally over all teams of the department (same CASE as the list).
	admissionQ := `
		SELECT adm_status, COUNT(*) AS cnt FROM (
			SELECT ` + fmt.Sprintf(admissionCaseSQL, "x") + ` AS adm_status
			FROM (
				SELECT
					pd.result AS pre_defense_result,
					adm.grade AS adm_grade,
					COALESCE(adm.passing_score, 0) AS adm_passing
				FROM teams t
				LEFT JOIN projects p ON p.team_id = t.id
				LEFT JOIN LATERAL (
					SELECT result FROM admin_pre_defense_submissions WHERE project_id = p.id ORDER BY created_at DESC LIMIT 1
				) pd ON TRUE
				LEFT JOIN LATERAL (
					SELECT g.grade, COALESCE((s2.config->'review_config'->>'passing_score')::int, 0) AS passing_score
					FROM admin_grades g JOIN states s2 ON s2.id = g.state_id
					WHERE g.project_id = p.id AND g.deleted_at IS NULL
					  AND (s2.config->'review_config'->>'grade_type') = 'admission'
					ORDER BY g.created_at DESC LIMIT 1
				) adm ON TRUE
				WHERE t.department_id = ? AND t.deleted_at IS NULL
			) x
		) y
		GROUP BY adm_status`
	type admRow struct {
		AdmStatus string `gorm:"column:adm_status"`
		Cnt       int64  `gorm:"column:cnt"`
	}
	var admRows []admRow
	if err := r.db.WithContext(ctx).Raw(admissionQ, departmentID).Scan(&admRows).Error; err != nil {
		return nil, err
	}
	for _, a := range admRows {
		switch a.AdmStatus {
		case AdmissionAdmitted:
			out.AdmittedCount = int32(a.Cnt)
		case AdmissionNotAdmitted:
			out.NotAdmittedCount = int32(a.Cnt)
		case AdmissionRevisionRequired:
			out.RevisionRequiredCount = int32(a.Cnt)
		}
	}

	var avg sql.NullFloat64
	if err := r.db.WithContext(ctx).
		Table("admin_grades g").
		Joins("JOIN projects p ON p.id = g.project_id").
		Where("p.department_id = ? AND g.deleted_at IS NULL", departmentID).
		Select("AVG(g.grade)").
		Scan(&avg).Error; err != nil {
		return nil, err
	}
	if avg.Valid {
		out.AverageGrade = float32(avg.Float64)
		out.HasAverageGrade = true
	}

	return out, nil
}
