package plugins

import (
	"fmt"
	"sync"
)

var globalRegistry = &Registry{
	plugins: make(map[string]ActionPlugin),
}

// Registry - глобальный реестр плагинов
type Registry struct {
	plugins map[string]ActionPlugin
	mu      sync.RWMutex
}

// Register регистрирует плагин глобально
func Register(plugin ActionPlugin) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.plugins[plugin.ID()] = plugin
}

// Get возвращает плагин по ID
func Get(id string) (ActionPlugin, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	if p, ok := globalRegistry.plugins[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("plugin not found: %s", id)
}

// MustGet возвращает плагин или паникует
func MustGet(id string) ActionPlugin {
	p, err := Get(id)
	if err != nil {
		panic(err)
	}
	return p
}

// List возвращает все зарегистрированные плагины
func List() []ActionPlugin {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make([]ActionPlugin, 0, len(globalRegistry.plugins))
	for _, p := range globalRegistry.plugins {
		result = append(result, p)
	}
	return result
}

// ListByCategory возвращает плагины по категории
func ListByCategory(category string) []ActionPlugin {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var result []ActionPlugin
	for _, p := range globalRegistry.plugins {
		if p.Category() == category {
			result = append(result, p)
		}
	}
	return result
}

// GetSchemas возвращает схемы всех плагинов для UI
func GetSchemas() []PluginSchema {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	schemas := make([]PluginSchema, 0, len(globalRegistry.plugins))
	for _, p := range globalRegistry.plugins {
		schemas = append(schemas, PluginSchema{
			ID:           p.ID(),
			Name:         p.Name(),
			Description:  p.Description(),
			Category:     p.Category(),
			ConfigSchema: p.ConfigSchema(),
			IsReversible: p.IsReversible(),
		})
	}
	return schemas
}

// GetSchemasByCategory возвращает схемы по категории
func GetSchemasByCategory(category string) []PluginSchema {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var schemas []PluginSchema
	for _, p := range globalRegistry.plugins {
		if p.Category() == category {
			schemas = append(schemas, PluginSchema{
				ID:           p.ID(),
				Name:         p.Name(),
				Description:  p.Description(),
				Category:     p.Category(),
				ConfigSchema: p.ConfigSchema(),
				IsReversible: p.IsReversible(),
			})
		}
	}
	return schemas
}

// Count возвращает количество зарегистрированных плагинов
func Count() int {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	return len(globalRegistry.plugins)
}

// Has проверяет наличие плагина
func Has(id string) bool {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	_, ok := globalRegistry.plugins[id]
	return ok
}
