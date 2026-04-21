package portaldb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyDepartmentMigrationSQLUsesBooleanExpressionForPostgres(t *testing.T) {
	statement := legacyDepartmentMigrationSQL("postgres")
	require.Contains(t, statement, "COALESCE(d.deleted, FALSE)")
	require.NotContains(t, statement, "COALESCE(d.deleted, 0) = 1")
}

func TestLegacyAISensitiveRuleMigrationSQLUsesBooleanExpressionForPostgres(t *testing.T) {
	detectStatement := legacyAISensitiveDetectRuleMigrationSQL("postgres")
	replaceStatement := legacyAISensitiveReplaceRuleMigrationSQL("postgres")

	require.Contains(t, detectStatement, "COALESCE(is_enabled, TRUE)")
	require.Contains(t, replaceStatement, "COALESCE(is_enabled, TRUE)")
	require.NotContains(t, detectStatement, "COALESCE(is_enabled, 1)")
	require.NotContains(t, replaceStatement, "COALESCE(is_enabled, 1)")
}
