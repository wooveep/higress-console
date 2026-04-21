package jobs

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

type retryingClient struct {
	db         *sql.DB
	ensureErrs []error
	ensureCall int
}

func (c *retryingClient) Healthy(ctx context.Context) error { return nil }
func (c *retryingClient) Enabled() bool                     { return true }
func (c *retryingClient) DB() *sql.DB                       { return c.db }
func (c *retryingClient) Driver() string                    { return "postgres" }
func (c *retryingClient) MigrateLegacyData(ctx context.Context) error {
	return nil
}

func (c *retryingClient) EnsureSchema(ctx context.Context) error {
	if c.ensureCall >= len(c.ensureErrs) {
		c.ensureCall++
		return nil
	}
	err := c.ensureErrs[c.ensureCall]
	c.ensureCall++
	return err
}

var _ portaldbclient.Client = (*retryingClient)(nil)

func TestDBRetriesSchemaCheckAfterFailure(t *testing.T) {
	service := New(&retryingClient{
		db:         &sql.DB{},
		ensureErrs: []error{errors.New("dial tcp postgres-server:5432: connect: connection refused"), nil},
	}, nil, nil, nil)

	_, err := service.db(context.Background())
	require.EqualError(t, err, "dial tcp postgres-server:5432: connect: connection refused")

	db, err := service.db(context.Background())
	require.NoError(t, err)
	require.NotNil(t, db)

	client := service.client.(*retryingClient)
	require.Equal(t, 2, client.ensureCall)
}
