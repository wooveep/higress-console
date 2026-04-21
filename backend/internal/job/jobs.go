package job

type Definition struct {
	Name        string
	Description string
	Schedule    string
	ManualOnly  bool
}

var PlannedJobs = []Definition{
	{
		Name:        "portal-consumer-projection",
		Description: "DB users and API keys -> key-auth projection",
		Schedule:    "0 */5 * * * *",
		ManualOnly:  false,
	},
	{
		Name:        "portal-consumer-level-auth-reconcile",
		Description: "Rebuild level-based allow-lists for route, AI route, and MCP",
		Schedule:    "15 */5 * * * *",
		ManualOnly:  false,
	},
	{
		Name:        "ai-sensitive-projection",
		Description: "Sync DB sensitive-word truth source to runtime plugin config",
		Schedule:    "30 */10 * * * *",
		ManualOnly:  false,
	},
	{
		Name:        "ai-model-rate-limit-reconcile",
		Description: "Project published model RPM/TPM limits into per-user per-model runtime rules",
		Schedule:    "40 */10 * * * *",
		ManualOnly:  false,
	},
	{
		Name:        "ai-plugin-execution-order-reconcile",
		Description: "Reconcile AI plugin execution order in runtime resources",
		Schedule:    "45 */10 * * * *",
		ManualOnly:  false,
	},
}
