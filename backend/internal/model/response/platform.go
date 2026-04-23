package response

type HealthzStatus struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	Phase         string `json:"phase"`
	LegacyBackend string `json:"legacyBackend"`
}

type SystemInfo struct {
	Service         string   `json:"service"`
	Version         string   `json:"version"`
	Phase           string   `json:"phase"`
	PreferredNaming string   `json:"preferredNaming"`
	LegacyNaming    string   `json:"legacyNaming"`
	LegacyBackend   string   `json:"legacyBackend"`
	BusinessLines   []string `json:"businessLines"`
}

type SystemConfigSnapshot struct {
	Module                string         `json:"module"`
	ServerAddress         string         `json:"serverAddress"`
	ExplicitRenameTargets []string       `json:"explicitRenameTargets"`
	ContractDirectories   []string       `json:"contractDirectories"`
	Properties            map[string]any `json:"properties,omitempty"`
}
