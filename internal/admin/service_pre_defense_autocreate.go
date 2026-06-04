package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type preDefenseSubmissionData struct {
	FileIDs []string `json:"file_ids"`
	Comment string   `json:"comment"`
}

func (s *Service) EnsurePreDefenseSubmissionForSubmission(ctx context.Context, sub *Submission) (*PreDefenseSubmission, error) {
	if sub == nil {
		return nil, nil
	}

	if sub.ProjectID <= 0 {
		return nil, fmt.Errorf("project_id is required for pre-defense auto-create")
	}

	if sub.TeamID <= 0 {
		return nil, fmt.Errorf("team_id is required for pre-defense auto-create")
	}

	isPreDefense, err := s.repo.IsPreDefenseState(ctx, sub.StepID)
	if err != nil {
		return nil, fmt.Errorf("check pre-defense state: %w", err)
	}

	if !isPreDefense {
		return nil, nil
	}

	now := time.Now()

	pd, err := s.repo.GetActivePreDefenseByProjectTeam(ctx, sub.ProjectID, sub.TeamID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get active pre-defense submission: %w", err)
		}

		pd = &PreDefenseSubmission{
			ID:              uuid.NewString(),
			TeamID:          sub.TeamID,
			ProjectID:       sub.ProjectID,
			SubmittedBy:     sub.SubmittedBy,
			Status:          StatusPending,
			DurationMinutes: 30,
			SubmittedAt:     now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := s.repo.CreatePreDefenseSubmission(ctx, pd); err != nil {
			return nil, fmt.Errorf("create pre-defense submission: %w", err)
		}

		_ = s.repo.AddPreDefenseHistory(ctx, &PreDefenseHistory{
			SubmissionID: pd.ID,
			Action:       "submitted",
			ActorID:      sub.SubmittedBy,
			OldValue:     "",
			NewValue:     StatusPending,
			Comment:      "Pre-defense submission was created automatically from workflow document submission",
			CreatedAt:    now,
		})
	}

	fileIDs := extractPreDefenseFileIDs(sub)
	for _, fileID := range fileIDs {
		exists, err := s.repo.PreDefenseDocumentExists(ctx, pd.ID, fileID)
		if err != nil {
			return nil, fmt.Errorf("check pre-defense document exists: %w", err)
		}
		if exists {
			continue
		}

		uploadedBy := sub.SubmittedBy
		doc := &PreDefenseDocument{
			ID:           fileID,
			SubmissionID: pd.ID,
			FileName:     fileID,
			FileType:     "",
			UploadedBy:   &uploadedBy,
			UploadedAt:   now,
			CreatedAt:    now,
		}

		if err := s.repo.AddPreDefenseDocument(ctx, doc); err != nil {
			return nil, fmt.Errorf("add pre-defense document: %w", err)
		}
	}

	return pd, nil
}

func extractPreDefenseFileIDs(sub *Submission) []string {
	if sub == nil {
		return nil
	}

	seen := make(map[string]struct{})
	out := make([]string, 0)

	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	if len(sub.Data) > 0 {
		var data preDefenseSubmissionData
		if err := json.Unmarshal(sub.Data, &data); err == nil {
			for _, id := range data.FileIDs {
				add(id)
			}
		}
	}

	if len(sub.Files) > 0 {
		var fileIDs []string
		if err := json.Unmarshal(sub.Files, &fileIDs); err == nil {
			for _, id := range fileIDs {
				add(id)
			}
		}

		var fileObjects []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(sub.Files, &fileObjects); err == nil {
			for _, f := range fileObjects {
				add(f.ID)
			}
		}
	}

	return out
}
