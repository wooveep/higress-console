package middleware

import "testing"

func TestAnonymousRequestMatching(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected bool
	}{
		{method: "GET", path: "/", expected: true},
		{method: "GET", path: "/healthz", expected: true},
		{method: "GET", path: "/healthz/ready", expected: true},
		{method: "GET", path: "/landing", expected: true},
		{method: "GET", path: "/login", expected: true},
		{method: "GET", path: "/init", expected: true},
		{method: "GET", path: "/dashboard", expected: true},
		{method: "GET", path: "/plugin", expected: true},
		{method: "GET", path: "/system", expected: true},
		{method: "GET", path: "/user/changePassword", expected: true},
		{method: "POST", path: "/user/changePassword", expected: false},
		{method: "GET", path: "/assets/index-BMsDsYbx.js", expected: true},
		{method: "GET", path: "/assets/index-AicemZYb.css", expected: true},
		{method: "GET", path: "/favicon.ico", expected: true},
		{method: "GET", path: "/logo-ai.svg", expected: true},
		{method: "GET", path: "/session/login", expected: true},
		{method: "GET", path: "/system/info", expected: true},
		{method: "GET", path: "/system/config", expected: true},
		{method: "GET", path: "/system/init", expected: true},
		{method: "GET", path: "/dashboard/info", expected: false},
		{method: "GET", path: "/user/info", expected: false},
		{method: "GET", path: "/v1/routes", expected: false},
		{method: "GET", path: "/mcp-templates/index", expected: true},
	}

	for _, tt := range tests {
		if actual := isAnonymousRequest(tt.method, tt.path); actual != tt.expected {
			t.Fatalf("%s %s anonymous=%v, want %v", tt.method, tt.path, actual, tt.expected)
		}
	}
}

func TestFrontendPathMatching(t *testing.T) {
	tests := map[string]bool{
		"/":                    true,
		"/plugin":              true,
		"/system":              true,
		"/user/changePassword": true,
		"/ai/dashboard":        true,
		"/dashboard/info":      false,
		"/user/info":           false,
		"/system/config":       false,
		"/v1/routes":           false,
		"/logo-ai.svg":         false,
	}

	for requestPath, expected := range tests {
		if actual := IsFrontendPagePath(requestPath); actual != expected {
			t.Fatalf("path %s page=%v, want %v", requestPath, actual, expected)
		}
	}
}

func TestFrontendStaticAssetMatching(t *testing.T) {
	tests := map[string]bool{
		"/logo-ai.svg":         true,
		"/banner.png":          true,
		"/avatar.png":          true,
		"/favicon.ico":         true,
		"/assets/index-123.js": true,
		"/system/config":       false,
		"/user/changePassword": false,
		"/v1/ai/routes":        false,
	}

	for requestPath, expected := range tests {
		if actual := IsFrontendStaticAssetPath(requestPath); actual != expected {
			t.Fatalf("path %s static=%v, want %v", requestPath, actual, expected)
		}
	}
}
