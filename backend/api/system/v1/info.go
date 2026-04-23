package v1

import "github.com/gogf/gf/v2/frame/g"

type InfoReq struct {
	g.Meta `path:"/system/info" tags:"System" method:"get" summary:"Return backend system info"`
}

type InfoRes struct {
	Service         string   `json:"service"`
	Version         string   `json:"version"`
	Phase           string   `json:"phase"`
	PreferredNaming string   `json:"preferredNaming"`
	LegacyNaming    string   `json:"legacyNaming"`
	LegacyBackend   string   `json:"legacyBackend"`
	BusinessLines   []string `json:"businessLines"`
}
