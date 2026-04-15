package consts

const (
	ServiceName                 = "aigateway-console"
	LegacyBackendDir            = "backend-java-legacy"
	CurrentPhase                = "P4-portal-domain"
	PreferredProduct            = "aigateway"
	LegacyProduct               = "higress"
	RequestIDHeader             = "X-Request-Id"
	AuthorizationHeader         = "Authorization"
	CtxKeyRequestID             = "request_id"
	CtxKeyCurrentUser           = "current_user"
	DefaultServerAddr           = ":8080"
	DefaultBuildVersion         = "0.1.0"
	DefaultAdminCookieName      = "_hi_sess"
	DefaultAdminCookieMaxAge    = 30 * 24 * 60 * 60
	DefaultAdminDisplayName     = "AIGateway Admin"
	DefaultAdminStateKey        = "default"
	DefaultConfigMapName        = "aigateway-console"
	DefaultHigressConfigMapName = "higress-config"
	DefaultHigressConfigDataKey = "higress"
	DefaultSecretName           = "aigateway-console"
	DefaultDashboardUIDMain     = "aigateway-main"
	DefaultDashboardUIDAI       = "aigateway-ai"
	DefaultDashboardUIDLog      = "aigateway-log"
	DefaultRouteName            = "default"
	DefaultDomainName           = "aigateway-default-domain"
	DefaultTLSCertificateName   = "default"
	DefaultTLSCertificateHost   = "aigateway-gateway"
	InternalResourceNameSuffix  = ".internal"
	DefaultConsoleServiceHost   = "aigateway-console.aigateway-system.svc.cluster.local"
	DefaultConsoleServicePort   = 8080
)
