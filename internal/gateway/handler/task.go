package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============== helpers ==============

func (h *Handler) resolveProjectIDForStudent(c *gin.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, errors.New("user_id is required")
	}

	// Prefer matching by team_id (more stable than "first project")
	teamResp, err := h.teamClient.GetMyTeam(outgoingCtx(c), &teamv1.GetMyTeamRequest{UserId: userID})
	if err != nil {
		return 0, err
	}
	if teamResp == nil || !teamResp.HasTeam || teamResp.Team == nil || teamResp.Team.TeamId == 0 {
		return 0, errors.New("user has no team")
	}
	teamID := teamResp.Team.TeamId

	projectsResp, err := h.projectClient.GetStudentProjects(outgoingCtx(c), &projectv1.GetStudentProjectsRequest{
		StudentId: userID,
	})
	if err != nil {
		return 0, err
	}
	if projectsResp == nil || len(projectsResp.Projects) == 0 {
		return 0, errors.New("no projects found for student")
	}

	for _, p := range projectsResp.Projects {
		if p != nil && p.TeamId == teamID && p.ProjectId > 0 {
			return p.ProjectId, nil
		}
	}

	// fallback: if only one project exists, return it
	if len(projectsResp.Projects) == 1 && projectsResp.Projects[0] != nil && projectsResp.Projects[0].ProjectId > 0 {
		return projectsResp.Projects[0].ProjectId, nil
	}

	return 0, errors.New("cannot resolve project for student")
}

func (h *Handler) resolveBoardID(c *gin.Context) (int64, error) {
	// 1) explicit board_id
	if b := c.Query("board_id"); b != "" {
		boardID, err := strconv.ParseInt(b, 10, 64)
		if err != nil || boardID <= 0 {
			return 0, errors.New("invalid board_id")
		}
		return boardID, nil
	}

	// 2) explicit project_id
	if p := c.Query("project_id"); p != "" {
		projectID, err := strconv.ParseInt(p, 10, 64)
		if err != nil || projectID <= 0 {
			return 0, errors.New("invalid project_id")
		}
		br, err := h.taskClient.GetBoardByProject(c.Request.Context(), &taskv1.GetBoardByProjectRequest{
			ProjectId: projectID,
		})
		if err != nil {
			return 0, err
		}
		if br == nil || br.Board == nil || br.Board.Id <= 0 {
			return 0, errors.New("board not found for project")
		}
		return br.Board.Id, nil
	}

	// 3) role-based fallback
	userID := c.GetInt64("userId")
	role := c.GetString("role")

	if userID <= 0 {
		return 0, errors.New("unauthorized")
	}

	switch role {
	case "student":
		projectID, err := h.resolveProjectIDForStudent(c, userID)
		if err != nil {
			return 0, err
		}
		br, err := h.taskClient.GetBoardByProject(c.Request.Context(), &taskv1.GetBoardByProjectRequest{
			ProjectId: projectID,
		})
		if err != nil {
			return 0, err
		}
		if br == nil || br.Board == nil || br.Board.Id <= 0 {
			return 0, errors.New("board not found")
		}
		return br.Board.Id, nil

	default: // teacher/admin/others
		lr, err := h.taskClient.ListMyBoards(c.Request.Context(), &taskv1.ListMyBoardsRequest{
			UserId: userID,
			Role:   role,
		})
		if err != nil {
			return 0, err
		}
		if lr == nil || len(lr.Boards) == 0 {
			return 0, errors.New("no boards found")
		}
		if len(lr.Boards) == 1 && lr.Boards[0] != nil && lr.Boards[0].Id > 0 {
			return lr.Boards[0].Id, nil
		}
		return 0, errors.New("multiple boards found; specify board_id or project_id")
	}
}

// ==================== Board ====================

func (h *Handler) GetBoard(c *gin.Context) {
	boardID, err := strconv.ParseInt(c.Param("board_id"), 10, 64)
	if err != nil || boardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	includeColumns := c.Query("include_columns") == "true"
	includeStats := c.Query("include_stats") == "true"
	includeTasks := c.Query("include_tasks") == "true"

	resp, err := h.taskClient.GetBoard(c.Request.Context(), &taskv1.GetBoardRequest{
		BoardId:        boardID,
		IncludeColumns: includeColumns,
		IncludeStats:   includeStats,
		IncludeTasks:   includeTasks,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/boards/project/:project_id
func (h *Handler) GetBoardByProject(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("project_id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	includeColumns := c.Query("include_columns") == "true"
	includeStats := c.Query("include_stats") == "true"
	includeTasks := c.Query("include_tasks") == "true"

	resp, err := h.taskClient.GetBoardByProject(c.Request.Context(), &taskv1.GetBoardByProjectRequest{
		ProjectId:      projectID,
		IncludeColumns: includeColumns,
		IncludeStats:   includeStats,
		IncludeTasks:   includeTasks,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/boards/my
func (h *Handler) ListMyBoards(c *gin.Context) {
	userID := c.GetInt64("userId")
	role := c.GetString("role")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	includeColumns := c.Query("include_columns") == "true"
	includeStats := c.Query("include_stats") == "true"
	resp, err := h.taskClient.ListMyBoards(taskCtx(c), &taskv1.ListMyBoardsRequest{
		UserId:         userID,
		Role:           role,
		IncludeColumns: includeColumns,
		IncludeStats:   includeStats,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateBoard(c *gin.Context) {
	boardID, err := strconv.ParseInt(c.Param("board_id"), 10, 64)
	if err != nil || boardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	var paths []string
	if req.Name != "" {
		paths = append(paths, "name")
	}
	if req.Description != "" {
		paths = append(paths, "description")
	}

	resp, err := h.taskClient.UpdateBoard(c.Request.Context(), &taskv1.UpdateBoardRequest{
		BoardId:     boardID,
		Name:        req.Name,
		Description: req.Description,
		UpdateMask:  &fieldmaskpb.FieldMask{Paths: paths},
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Columns ====================

func (h *Handler) ListColumns(c *gin.Context) {
	boardID, err := strconv.ParseInt(c.Param("board_id"), 10, 64)
	if err != nil || boardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	includeTaskCount := c.Query("include_task_count") == "true"

	resp, err := h.taskClient.ListColumns(c.Request.Context(), &taskv1.ListColumnsRequest{
		BoardId:          boardID,
		IncludeTaskCount: includeTaskCount,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateColumn(c *gin.Context) {
	boardID, err := strconv.ParseInt(c.Param("board_id"), 10, 64)
	if err != nil || boardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	var req struct {
		Name         string `json:"name" binding:"required"`
		Slug         string `json:"slug" binding:"required"`
		Description  string `json:"description"`
		Color        string `json:"color"`
		Icon         string `json:"icon"`
		WipLimit     int32  `json:"wip_limit"`
		IsDoneColumn bool   `json:"is_done_column"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.taskClient.CreateColumn(c.Request.Context(), &taskv1.CreateColumnRequest{
		BoardId:      boardID,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		Color:        req.Color,
		Icon:         req.Icon,
		WipLimit:     req.WipLimit,
		IsDoneColumn: req.IsDoneColumn,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) UpdateColumn(c *gin.Context) {
	columnID, err := strconv.ParseInt(c.Param("column_id"), 10, 64)
	if err != nil || columnID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid column id"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
		Icon        string `json:"icon"`
		WipLimit    int32  `json:"wip_limit"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.taskClient.UpdateColumn(c.Request.Context(), &taskv1.UpdateColumnRequest{
		ColumnId:    columnID,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		WipLimit:    req.WipLimit,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteColumn(c *gin.Context) {
	columnID, err := strconv.ParseInt(c.Param("column_id"), 10, 64)
	if err != nil || columnID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid column id"})
		return
	}

	var moveToColumnID int64
	if m := c.Query("move_tasks_to"); m != "" {
		moveToColumnID, _ = strconv.ParseInt(m, 10, 64)
	}

	_, err = h.taskClient.DeleteColumn(c.Request.Context(), &taskv1.DeleteColumnRequest{
		ColumnId:            columnID,
		MoveTasksToColumnId: moveToColumnID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ReorderColumns(c *gin.Context) {
	boardID, err := strconv.ParseInt(c.Param("board_id"), 10, 64)
	if err != nil || boardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	var req struct {
		ColumnIDs []int64 `json:"column_ids" binding:"required"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.taskClient.ReorderColumns(c.Request.Context(), &taskv1.ReorderColumnsRequest{
		BoardId:   boardID,
		ColumnIds: req.ColumnIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Tasks ====================

func (h *Handler) CreateTask(c *gin.Context) {
	userID := c.GetInt64("userId")

	var req struct {
		BoardID        int64    `json:"board_id" binding:"required"`
		Title          string   `json:"title" binding:"required"`
		Description    string   `json:"description"`
		Priority       string   `json:"priority"`
		AssigneeID     int64    `json:"assignee_id"`
		DueDate        string   `json:"due_date"` // ISO 8601 format: "2025-02-15T10:00:00Z"
		EstimatedMins  int32    `json:"estimated_minutes"`
		Labels         []string `json:"labels"`
		ColumnID       int64    `json:"column_id"`
		WorkflowStepID int64    `json:"workflow_step_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priority := taskv1.TaskPriority_TASK_PRIORITY_MEDIUM
	switch req.Priority {
	case "low":
		priority = taskv1.TaskPriority_TASK_PRIORITY_LOW
	case "high":
		priority = taskv1.TaskPriority_TASK_PRIORITY_HIGH
	case "urgent":
		priority = taskv1.TaskPriority_TASK_PRIORITY_URGENT
	}

	grpcReq := &taskv1.CreateTaskRequest{
		BoardId:          req.BoardID,
		Title:            req.Title,
		Description:      req.Description,
		Priority:         priority,
		AssigneeId:       req.AssigneeID,
		EstimatedMinutes: req.EstimatedMins,
		Labels:           req.Labels,
		ColumnId:         req.ColumnID,
		CreatedBy:        userID,
		WorkflowStepId:   req.WorkflowStepID,
	}

	// Parse DueDate if provided
	if req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, req.DueDate)
		if err == nil {
			grpcReq.DueDate = timestamppb.New(t)
		}
	}

	resp, err := h.taskClient.CreateTask(c.Request.Context(), grpcReq)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) GetTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	resp, err := h.taskClient.GetTask(c.Request.Context(), &taskv1.GetTaskRequest{
		TaskId: taskID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	userID := c.GetInt64("userId")

	var req struct {
		Title          string   `json:"title"`
		Description    string   `json:"description"`
		Priority       string   `json:"priority"`
		EstimatedMins  int32    `json:"estimated_minutes"`
		ActualMins     int32    `json:"actual_minutes"`
		Labels         []string `json:"labels"`
		WorkflowStepID int64    `json:"workflow_step_id"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	priority := taskv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED
	switch req.Priority {
	case "low":
		priority = taskv1.TaskPriority_TASK_PRIORITY_LOW
	case "medium":
		priority = taskv1.TaskPriority_TASK_PRIORITY_MEDIUM
	case "high":
		priority = taskv1.TaskPriority_TASK_PRIORITY_HIGH
	case "urgent":
		priority = taskv1.TaskPriority_TASK_PRIORITY_URGENT
	}

	resp, err := h.taskClient.UpdateTask(c.Request.Context(), &taskv1.UpdateTaskRequest{
		TaskId:           taskID,
		Title:            req.Title,
		Description:      req.Description,
		Priority:         priority,
		EstimatedMinutes: req.EstimatedMins,
		ActualMinutes:    req.ActualMins,
		Labels:           req.Labels,
		WorkflowStepId:   req.WorkflowStepID,
		UpdatedBy:        userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	userID := c.GetInt64("userId")

	_, err = h.taskClient.DeleteTask(c.Request.Context(), &taskv1.DeleteTaskRequest{
		TaskId:    taskID,
		DeletedBy: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ListTasks(c *gin.Context) {
	var boardID, columnID, assigneeID int64
	if b := c.Query("board_id"); b != "" {
		boardID, _ = strconv.ParseInt(b, 10, 64)
	}
	if col := c.Query("column_id"); col != "" {
		columnID, _ = strconv.ParseInt(col, 10, 64)
	}
	if a := c.Query("assignee_id"); a != "" {
		assigneeID, _ = strconv.ParseInt(a, 10, 64)
	}

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, _ := strconv.ParseInt(ps, 10, 32); v > 0 {
			pageSize = int32(v)
		}
	}

	resp, err := h.taskClient.ListTasks(c.Request.Context(), &taskv1.ListTasksRequest{
		BoardId:     boardID,
		ColumnId:    columnID,
		AssigneeId:  assigneeID,
		Search:      c.Query("search"),
		OnlyOverdue: c.Query("only_overdue") == "true",
		SortBy:      c.Query("sort_by"),
		SortOrder:   c.Query("sort_order"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Kanban Operations ====================

func (h *Handler) MoveTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	userID := c.GetInt64("userId")

	var req struct {
		ToColumnID int64 `json:"to_column_id" binding:"required"`
		Position   int32 `json:"position"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.taskClient.MoveTask(c.Request.Context(), &taskv1.MoveTaskRequest{
		TaskId:     taskID,
		ToColumnId: req.ToColumnID,
		Position:   req.Position,
		MovedBy:    userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReorderTasks(c *gin.Context) {
	columnID, err := strconv.ParseInt(c.Param("column_id"), 10, 64)
	if err != nil || columnID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid column id"})
		return
	}

	var req struct {
		TaskIDs []int64 `json:"task_ids" binding:"required"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	_, err = h.taskClient.ReorderTasks(c.Request.Context(), &taskv1.ReorderTasksRequest{
		ColumnId: columnID,
		TaskIds:  req.TaskIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ==================== Assignment ====================

func (h *Handler) AssignTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	userID := c.GetInt64("userId")

	var req struct {
		AssigneeID int64 `json:"assignee_id" binding:"required"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.taskClient.AssignTask(c.Request.Context(), &taskv1.AssignTaskRequest{
		TaskId:     taskID,
		AssigneeId: req.AssigneeID,
		AssignedBy: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UnassignTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	userID := c.GetInt64("userId")

	resp, err := h.taskClient.UnassignTask(c.Request.Context(), &taskv1.UnassignTaskRequest{
		TaskId:       taskID,
		UnassignedBy: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Comments ====================

func (h *Handler) CreateTaskComment(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	userID := c.GetInt64("userId")

	var req struct {
		Content        string  `json:"content" binding:"required"`
		MentionUserIDs []int64 `json:"mention_user_ids"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	resp, err := h.taskClient.CreateComment(c.Request.Context(), &taskv1.CreateCommentRequest{
		TaskId:         taskID,
		AuthorId:       userID,
		Content:        req.Content,
		MentionUserIds: req.MentionUserIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) ListTaskComments(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}

	resp, err := h.taskClient.ListComments(c.Request.Context(), &taskv1.ListCommentsRequest{
		TaskId:   taskID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteTaskComment(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil || commentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}
	userID := c.GetInt64("userId")

	_, err = h.taskClient.DeleteComment(c.Request.Context(), &taskv1.DeleteCommentRequest{
		CommentId: commentID,
		DeletedBy: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// ==================== Dashboard & Stats ====================

func (h *Handler) GetBoardStats(c *gin.Context) {
	boardID, err := strconv.ParseInt(c.Param("board_id"), 10, 64)
	if err != nil || boardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	includeMemberStats := c.Query("include_member_stats") == "true"
	includeDailyStats := c.Query("include_daily_stats") == "true"

	resp, err := h.taskClient.GetBoardStats(c.Request.Context(), &taskv1.GetBoardStatsRequest{
		BoardId:            boardID,
		IncludeMemberStats: includeMemberStats,
		IncludeDailyStats:  includeDailyStats,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetMyTasks(c *gin.Context) {
	userID := c.GetInt64("userId")

	role := c.GetString("role")
	if role == "" {
		role = "user"
	}
	if role == "user" {
		role = "student"
	}

	ctx := metadata.NewOutgoingContext(
		c.Request.Context(),
		metadata.Pairs(
			"x-user-id", strconv.FormatInt(userID, 10),
			"x-user-role", role,
			"x-university-id", strconv.FormatInt(c.GetInt64("universityId"), 10),
			"x-department-id", strconv.FormatInt(c.GetInt64("departmentId"), 10),
		),
	)

	resp, err := h.taskClient.GetMyTasks(ctx, &taskv1.GetMyTasksRequest{
		OnlyAssigned: c.Query("only_assigned") == "true",
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetOverdueTasks(c *gin.Context) {
	var boardID, assigneeID int64
	if b := c.Query("board_id"); b != "" {
		boardID, _ = strconv.ParseInt(b, 10, 64)
	}
	if a := c.Query("assignee_id"); a != "" {
		assigneeID, _ = strconv.ParseInt(a, 10, 64)
	}

	page := int32(1)
	pageSize := int32(20)

	resp, err := h.taskClient.GetOverdueTasks(c.Request.Context(), &taskv1.GetOverdueTasksRequest{
		BoardId:    boardID,
		AssigneeId: assigneeID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetUpcomingDeadlines(c *gin.Context) {
	userID := c.GetInt64("userId")

	var boardID int64
	if b := c.Query("board_id"); b != "" {
		boardID, _ = strconv.ParseInt(b, 10, 64)
	}

	daysAhead := int32(7)
	if d := c.Query("days_ahead"); d != "" {
		if v, _ := strconv.ParseInt(d, 10, 32); v > 0 {
			daysAhead = int32(v)
		}
	}

	if boardID == 0 {
		// Try to resolve board automatically:
		// - use board_id or project_id if provided,
		// - else by role fallback (student: team->project->board; teacher: list my boards)
		bid, err := h.resolveBoardID(c)
		if err != nil {
			// keep old behavior: return empty list instead of hard error
			c.JSON(http.StatusOK, &taskv1.ListTasksResponse{
				Tasks:      nil,
				TotalCount: 0,
			})
			return
		}
		boardID = bid
	}

	resp, err := h.taskClient.GetUpcomingDeadlines(c.Request.Context(), &taskv1.GetUpcomingDeadlinesRequest{
		BoardId:   boardID,
		UserId:    userID,
		DaysAhead: daysAhead,
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Activity ====================

func (h *Handler) GetTaskActivity(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	page := int32(1)
	pageSize := int32(20)

	resp, err := h.taskClient.GetTaskActivity(c.Request.Context(), &taskv1.GetTaskActivityRequest{
		TaskId:   taskID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ==================== Watchers ====================

func (h *Handler) AddTaskWatcher(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	_, err = h.taskClient.AddWatcher(c.Request.Context(), &taskv1.AddWatcherRequest{
		TaskId: taskID,
		UserId: req.UserID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) RemoveTaskWatcher(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	_, err = h.taskClient.RemoveWatcher(c.Request.Context(), &taskv1.RemoveWatcherRequest{
		TaskId: taskID,
		UserId: req.UserID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ListTaskWatchers(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	resp, err := h.taskClient.ListWatchers(c.Request.Context(), &taskv1.ListWatchersRequest{
		TaskId: taskID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
