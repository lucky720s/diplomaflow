package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ginInt64(c *gin.Context, key string) int64 {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	default:
		return 0
	}
}

func ginString(c *gin.Context, key string) string {
	v, ok := c.Get(key)
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func onboardingCtx(c *gin.Context) context.Context {
	userID := ginInt64(c, "userId")
	role := ginString(c, "role")
	universityID := ginInt64(c, "universityId")
	departmentID := ginInt64(c, "departmentId")
	traceID := c.GetString("trace_id")

	return metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", role,
		"x-university-id", strconv.FormatInt(universityID, 10),
		"x-department-id", strconv.FormatInt(departmentID, 10),
		"x-trace-id", traceID,
	)
}

func (h *Handler) GetOnboardingStatus(c *gin.Context) {
	userID := ginInt64(c, "userId")
	departmentID := ginInt64(c, "departmentId")
	role := ginString(c, "role")

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	wf, err := h.workflowClient.GetActiveWorkflowByDepartment(c.Request.Context(), &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			c.JSON(http.StatusOK, gin.H{
				"stage":         "NO_WORKFLOW",
				"has_workflow":  false,
				"workflow":      nil,
				"department_id": departmentID,
				"user_id":       userID,
				"role":          role,
				"message":       "No active workflow for this department",
			})
			return
		}
		MapGRPCError(c, err)
		return
	}

	if wf == nil || wf.Id == 0 {
		c.JSON(http.StatusOK, gin.H{
			"stage":         "NO_WORKFLOW",
			"has_workflow":  false,
			"workflow":      nil,
			"department_id": departmentID,
			"user_id":       userID,
			"role":          role,
			"message":       "No active workflow for this department",
		})
		return
	}

	ctx := onboardingCtx(c)

	teamResp, err := h.teamClient.GetMyTeam(ctx, &teamv1.GetMyTeamRequest{UserId: userID})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	teamCfg, _ := h.workflowClient.GetTeamConfiguration(c.Request.Context(), &workflowv1.GetTeamConfigurationRequest{
		DepartmentId: departmentID,
		WorkflowId:   wf.Id,
		StateId:      0,
	})

	if teamResp == nil || !teamResp.HasTeam || teamResp.Team == nil || teamResp.Team.TeamId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"stage":              "TEAM_FORMATION",
			"has_workflow":       true,
			"workflow":           wf,
			"department_id":      departmentID,
			"user_id":            userID,
			"role":               role,
			"has_team":           false,
			"team":               nil,
			"team_configuration": teamCfg,
			"has_project":        false,
			"project":            nil,
		})
		return
	}

	teamID := teamResp.Team.TeamId

	projectsResp, err := h.projectClient.GetStudentProjects(ctx, &projectv1.GetStudentProjectsRequest{
		StudentId: userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	var matched *projectv1.ProjectPreview
	if projectsResp != nil {
		for _, p := range projectsResp.Projects {
			if p != nil && p.TeamId == teamID {
				matched = p
				break
			}
		}
	}

	if matched == nil {
		c.JSON(http.StatusOK, gin.H{
			"stage":              "SUPERVISOR_SELECTION",
			"has_workflow":       true,
			"workflow":           wf,
			"department_id":      departmentID,
			"user_id":            userID,
			"role":               role,
			"has_team":           true,
			"team":               teamResp.Team,
			"team_configuration": teamCfg,
			"has_project":        false,
			"project":            nil,
		})
		return
	}

	stage := matched.CurrentStateName
	if stage == "" {
		stage = matched.CurrentState
	}

	c.JSON(http.StatusOK, gin.H{
		"stage":              stage,
		"has_workflow":       true,
		"workflow":           wf,
		"department_id":      departmentID,
		"user_id":            userID,
		"role":               role,
		"has_team":           true,
		"team":               teamResp.Team,
		"team_configuration": teamCfg,
		"has_project":        true,
		"project":            matched,
	})
}
