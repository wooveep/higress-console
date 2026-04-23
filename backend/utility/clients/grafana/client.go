package grafana

import "context"

type Config struct {
	Enabled  bool
	BaseURL  string
	Username string
	Password string
}

type Client interface {
	Healthy(ctx context.Context) error
	IsBuiltIn() bool
	BaseURL() string
}

type FakeClient struct {
	config Config
}

func New(cfg Config) Client {
	return &FakeClient{config: cfg}
}

func (c *FakeClient) Healthy(ctx context.Context) error { return nil }
func (c *FakeClient) IsBuiltIn() bool                   { return c.config.Enabled && c.config.BaseURL != "" }
func (c *FakeClient) BaseURL() string                   { return c.config.BaseURL }
