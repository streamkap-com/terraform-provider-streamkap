package helper

import (
	"context"
	"testing"
)

func TestMigrateDeprecatedValues(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		config   map[string]any
		aliases  []DeprecatedAlias
		expected map[string]any
	}{
		{
			name: "migrates deprecated value when new value not set",
			config: map[string]any{
				"old_name": "test_value",
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"new_name": "test_value",
			},
		},
		{
			name: "does not overwrite existing new value",
			config: map[string]any{
				"old_name": "old_value",
				"new_name": "new_value",
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"new_name": "new_value",
			},
		},
		{
			name: "handles nil deprecated value",
			config: map[string]any{
				"old_name": nil,
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"old_name": nil,
			},
		},
		{
			name: "handles empty deprecated value",
			config: map[string]any{
				"old_name": "",
			},
			aliases: []DeprecatedAlias{
				{OldName: "old_name", NewName: "new_name"},
			},
			expected: map[string]any{
				"old_name": "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MigrateDeprecatedValues(ctx, tc.config, tc.aliases)
			for key, expectedVal := range tc.expected {
				if result[key] != expectedVal {
					t.Errorf("expected %s=%v, got %v", key, expectedVal, result[key])
				}
			}
		})
	}
}

func TestPostgreSQLDeprecatedAliases(t *testing.T) {
	if len(PostgreSQLDeprecatedAliases) != 9 {
		t.Errorf("expected 9 PostgreSQL deprecated aliases, got %d", len(PostgreSQLDeprecatedAliases))
	}
}

func TestSnowflakeDeprecatedAliases(t *testing.T) {
	if len(SnowflakeDeprecatedAliases) != 1 {
		t.Errorf("expected 1 Snowflake deprecated alias, got %d", len(SnowflakeDeprecatedAliases))
	}
}
