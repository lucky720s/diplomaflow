package team

import (
	"context"
	"encoding/json"
	"log"

	"github.com/lucky720s/diplomaflow/pkg/broker"
)

type EventHandler struct {
	service *Service
}

func NewEventHandler(service *Service) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) HandleProjectCreated(event broker.Event) error {
	log.Println("Processing ProjectCreated event...")

	data, err := json.Marshal(event.Payload)
	if err != nil {
		log.Printf("Error marshaling payload: %v", err)
		return nil
	}

	var payload struct {
		ProjectID float64 `json:"project_id"`
		Title     string  `json:"title"`
		StudentID float64 `json:"student_id"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("Error unmarshaling payload: %v", err)
		return nil
	}

	memberIDs := []int64{int64(payload.StudentID)}
	teamID, err := h.service.CreateTeam(context.Background(), payload.Title, int64(payload.ProjectID), memberIDs)
	if err != nil {
		log.Printf("Failed to create team: %v", err)
		return err
	}

	log.Printf("Successfully created Team ID %d for Project ID %v", teamID, payload.ProjectID)
	return nil
}
