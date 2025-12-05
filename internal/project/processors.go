package project

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"gorm.io/datatypes"
)

type ActionHandler interface {
	Handle(ctx context.Context, currentData map[string]interface{}, actionPayload map[string]interface{}, stepConfig map[string]interface{}) (newData map[string]interface{}, err error)
}

type ProcessorRegistry struct {
	handlers map[string]ActionHandler
	mu       sync.RWMutex
}

func NewProcessorRegistry() *ProcessorRegistry {
	return &ProcessorRegistry{
		handlers: make(map[string]ActionHandler),
	}
}

func (r *ProcessorRegistry) Register(actionType string, handler ActionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[actionType] = handler
}

func (r *ProcessorRegistry) Get(actionType string) (ActionHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[actionType]
	if !ok {
		return nil, fmt.Errorf("no handler registered for action type: %s", actionType)
	}
	return handler, nil
}

func JSONToMap(src datatypes.JSON) (map[string]interface{}, error) {
	var dest map[string]interface{}
	if len(src) == 0 {
		return make(map[string]interface{}), nil
	}
	if err := json.Unmarshal(src, &dest); err != nil {
		return nil, err
	}
	return dest, nil
}

func MapToJSON(src map[string]interface{}) (datatypes.JSON, error) {
	bytes, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(bytes), nil
}
