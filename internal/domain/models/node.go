package models

type Node struct {
	Name           string
	HealthCheckURL string
	StartUpCommand string
	StartUpDir     string
	EnvVarPort     string
	OtherEnvVars   map[string]string
}
