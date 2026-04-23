package v1

import "github.com/gogf/gf/v2/frame/g"

type PingReq struct {
	g.Meta `path:"/healthz" tags:"Base" method:"get" summary:"Health check endpoint"`
}

type PingRes struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	Phase         string `json:"phase"`
	LegacyBackend string `json:"legacyBackend"`
}
