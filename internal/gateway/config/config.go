package config

import "os"

type Config struct {
	Port                  string
	JWTSecret             string
	AuthServiceAddr       string
	ProjectServiceAddr    string
	TeamServiceAddr       string
	WorkflowServiceAddr   string
	UniversityServiceAddr string
	RoleServiceAddr       string
}

func Load() *Config {
	return &Config{
		Port:                  getEnv("PORT", "8080"),
		JWTSecret:             getEnv("JWT_SECRET", "secret_key_change_me"),
		AuthServiceAddr:       getEnv("AUTH_SERVICE_ADDR", "auth_service:8082"),
		ProjectServiceAddr:    getEnv("PROJECT_SERVICE_ADDR", "project_service:8083"),
		TeamServiceAddr:       getEnv("TEAM_SERVICE_ADDR", "team_service:8084"),
		WorkflowServiceAddr:   getEnv("WORKFLOW_SERVICE_ADDR", "workflow_service:8085"),
		UniversityServiceAddr: getEnv("UNIVERSITY_SERVICE_ADDR", "university_service:8081"),
		RoleServiceAddr:       getEnv("ROLE_SERVICE_ADDR", "role_service:8086"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
