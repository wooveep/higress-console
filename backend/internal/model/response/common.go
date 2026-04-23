package response

type User struct {
	Name        string `json:"name,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Type        string `json:"type,omitempty"`
}

type DashboardInfo struct {
	BuiltIn bool   `json:"builtIn"`
	UID     string `json:"uid,omitempty"`
	URL     string `json:"url"`
}

type NativeDashboardVariableState struct {
	Value   string   `json:"value"`
	Options []string `json:"options"`
}

type NativeDashboardGridPos struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

type NativeDashboardPoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

type NativeDashboardSeries struct {
	Name   string                 `json:"name"`
	Labels map[string]string      `json:"labels"`
	Points []NativeDashboardPoint `json:"points"`
}

type NativeDashboardStat struct {
	Value *float64 `json:"value,omitempty"`
}

type NativeDashboardTableColumn struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

type NativeDashboardTable struct {
	Columns []NativeDashboardTableColumn `json:"columns"`
	Rows    []map[string]any             `json:"rows"`
}

type NativeDashboardPanel struct {
	ID      int                     `json:"id"`
	Title   string                  `json:"title"`
	Type    string                  `json:"type"`
	Unit    string                  `json:"unit"`
	GridPos NativeDashboardGridPos  `json:"gridPos"`
	Error   string                  `json:"error,omitempty"`
	Stat    *NativeDashboardStat    `json:"stat,omitempty"`
	Series  []NativeDashboardSeries `json:"series,omitempty"`
	Table   *NativeDashboardTable   `json:"table,omitempty"`
}

type NativeDashboardRow struct {
	Title     string                 `json:"title"`
	Collapsed bool                   `json:"collapsed"`
	Panels    []NativeDashboardPanel `json:"panels"`
}

type NativeDashboardData struct {
	Title          string `json:"title"`
	Type           string `json:"type"`
	From           int64  `json:"from"`
	To             int64  `json:"to"`
	DefaultRangeMS int64  `json:"defaultRangeMs"`
	Variables      struct {
		Gateway   NativeDashboardVariableState `json:"gateway"`
		Namespace NativeDashboardVariableState `json:"namespace"`
	} `json:"variables"`
	Rows []NativeDashboardRow `json:"rows"`
}
