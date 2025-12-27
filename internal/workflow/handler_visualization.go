package workflow

import (
	"context"
	"fmt"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

func (h *Handler) GetWorkflowVisualization(ctx context.Context, req *workflowv1.GetWorkflowVisualizationRequest) (*workflowv1.WorkflowVisualization, error) {
	wf, err := h.service.GetWorkflowFull(ctx, req.WorkflowId)
	if err != nil {
		return nil, err
	}

	var nodes []*workflowv1.VisualizationNode
	var edges []*workflowv1.VisualizationEdge

	// ИСПРАВЛЕНО: было wf.Steps, стало wf.States
	for _, state := range wf.States {
		nodes = append(nodes, &workflowv1.VisualizationNode{
			Id:    fmt.Sprint(state.ID),
			Label: state.DisplayName,
			Type:  state.Type,
			Color: state.Color,
			Position: &workflowv1.Position{
				X: float32(state.OrderIndex) * 200,
				Y: 100,
			},
			Data: map[string]string{
				"duration":   fmt.Sprintf("%d дней", state.DurationDays),
				"is_initial": fmt.Sprint(state.IsInitial),
				"is_final":   fmt.Sprint(state.IsFinal),
			},
		})
	}

	// ИСПРАВЛЕНО: используем wf.Transitions напрямую
	for _, tr := range wf.Transitions {
		edges = append(edges, &workflowv1.VisualizationEdge{
			Id:       fmt.Sprint(tr.ID),
			Source:   fmt.Sprint(tr.FromStateID),
			Target:   fmt.Sprint(tr.ToStateID),
			Label:    tr.DisplayName,
			Type:     "smoothstep",
			Animated: tr.ButtonColor == "danger",
		})
	}

	return &workflowv1.WorkflowVisualization{
		Nodes: nodes,
		Edges: edges,
	}, nil
}
