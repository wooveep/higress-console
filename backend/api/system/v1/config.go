package v1

import "github.com/gogf/gf/v2/frame/g"

type ConfigReq struct {
	g.Meta `path:"/system/config" tags:"System" method:"get" summary:"Return migration-phase config summary"`
}

type ConfigRes struct {
	Module                string         `json:"module"`
	ServerAddress         string         `json:"serverAddress"`
	ExplicitRenameTargets []string       `json:"explicitRenameTargets"`
	ContractDirectories   []string       `json:"contractDirectories"`
	Properties            map[string]any `json:"properties,omitempty"`
}
