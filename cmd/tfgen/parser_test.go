package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConnectorConfig_PostgreSQL(t *testing.T) {
	// This test requires access to the backend repository
	backendPath := "/Users/alexandrubodea/Documents/Repositories/python-be-streamkap"
	configPath := filepath.Join(backendPath, "app/sources/plugins/postgresql/configuration.latest.json")

	// Skip if backend is not available
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Backend repository not available, skipping test")
	}

	config, err := ParseConnectorConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to parse PostgreSQL config: %v", err)
	}

	// Verify basic structure
	if config.DisplayName != "PostgreSQL" {
		t.Errorf("Expected display_name 'PostgreSQL', got '%s'", config.DisplayName)
	}

	if len(config.Config) == 0 {
		t.Error("Expected non-empty config array")
	}

	// Verify user-defined entries
	userDefined := config.UserDefinedEntries()
	if len(userDefined) == 0 {
		t.Error("Expected non-empty user-defined entries")
	}

	// Verify specific entry: database.hostname.user.defined
	entry := config.GetEntryByName("database.hostname.user.defined")
	if entry == nil {
		t.Fatal("Expected to find 'database.hostname.user.defined' entry")
	}

	if !entry.IsUserDefined() {
		t.Error("Expected database.hostname.user.defined to be user-defined")
	}

	if entry.TerraformAttributeName() != "database_hostname" {
		t.Errorf("Expected terraform name 'database_hostname', got '%s'", entry.TerraformAttributeName())
	}

	if entry.TerraformType() != TerraformTypeString {
		t.Errorf("Expected terraform type String, got '%s'", entry.TerraformType())
	}

	// Verify password entry is sensitive
	passwordEntry := config.GetEntryByName("database.password")
	if passwordEntry == nil {
		t.Fatal("Expected to find 'database.password' entry")
	}

	if !passwordEntry.IsSensitive() {
		t.Error("Expected database.password to be sensitive")
	}

	// Verify slider entry
	sliderEntry := config.GetEntryByName("streamkap.snapshot.parallelism")
	if sliderEntry == nil {
		t.Fatal("Expected to find 'streamkap.snapshot.parallelism' entry")
	}

	if sliderEntry.TerraformType() != TerraformTypeInt64 {
		t.Errorf("Expected terraform type Int64 for slider, got '%s'", sliderEntry.TerraformType())
	}

	if sliderEntry.GetSliderMin() != 1 {
		t.Errorf("Expected slider min 1, got %d", sliderEntry.GetSliderMin())
	}

	if sliderEntry.GetSliderMax() != 10 {
		t.Errorf("Expected slider max 10, got %d", sliderEntry.GetSliderMax())
	}

	// Verify one-select entry with raw_values
	selectEntry := config.GetEntryByName("snapshot.read.only.user.defined")
	if selectEntry == nil {
		t.Fatal("Expected to find 'snapshot.read.only.user.defined' entry")
	}

	rawValues := selectEntry.GetRawValues()
	if len(rawValues) != 2 || rawValues[0] != "Yes" || rawValues[1] != "No" {
		t.Errorf("Expected raw_values ['Yes', 'No'], got %v", rawValues)
	}

	// Verify conditional entry
	conditionalEntry := config.GetEntryByName("ssh.host")
	if conditionalEntry == nil {
		t.Fatal("Expected to find 'ssh.host' entry")
	}

	if !conditionalEntry.IsConditional() {
		t.Error("Expected ssh.host to have conditions")
	}

	if len(conditionalEntry.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(conditionalEntry.Conditions))
	}

	cond := conditionalEntry.Conditions[0]
	if cond.Operator != "EQ" {
		t.Errorf("Expected condition operator 'EQ', got '%s'", cond.Operator)
	}
	if cond.Config != "ssh.enabled" {
		t.Errorf("Expected condition config 'ssh.enabled', got '%s'", cond.Config)
	}
	if !cond.GetConditionValueBool() {
		t.Error("Expected condition value to be true")
	}
}

func TestParseConnectorConfig_Snowflake(t *testing.T) {
	// This test requires access to the backend repository
	backendPath := "/Users/alexandrubodea/Documents/Repositories/python-be-streamkap"
	configPath := filepath.Join(backendPath, "app/destinations/plugins/snowflake/configuration.latest.json")

	// Skip if backend is not available
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Backend repository not available, skipping test")
	}

	config, err := ParseConnectorConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to parse Snowflake config: %v", err)
	}

	// Verify basic structure
	if config.DisplayName != "Snowflake" {
		t.Errorf("Expected display_name 'Snowflake', got '%s'", config.DisplayName)
	}

	// Verify set_once field
	ingestionMode := config.GetEntryByName("ingestion.mode")
	if ingestionMode == nil {
		t.Fatal("Expected to find 'ingestion.mode' entry")
	}

	if !ingestionMode.IsSetOnce() {
		t.Error("Expected ingestion.mode to be set_once")
	}

	// Verify sensitive field
	privateKey := config.GetEntryByName("snowflake.private.key")
	if privateKey == nil {
		t.Fatal("Expected to find 'snowflake.private.key' entry")
	}

	if !privateKey.IsSensitive() {
		t.Error("Expected snowflake.private.key to be sensitive")
	}

	// Verify boolean control
	boolEntry := config.GetEntryByName("snowflake.private.key.passphrase.secured")
	if boolEntry == nil {
		t.Fatal("Expected to find 'snowflake.private.key.passphrase.secured' entry")
	}

	if boolEntry.TerraformType() != TerraformTypeBool {
		t.Errorf("Expected terraform type Bool, got '%s'", boolEntry.TerraformType())
	}

	if !boolEntry.GetDefaultBool() {
		t.Error("Expected default value true for boolean field")
	}
}

func TestTerraformAttributeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"database.hostname.user.defined", "database_hostname"},
		{"database.port.user.defined", "database_port"},
		{"ssh.public.key.user.displayed", "ssh_public_key"},
		{"snowflake.url.name", "snowflake_url_name"},
		{"tasks.max", "tasks_max"},
		{"ingestion.mode", "ingestion_mode"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		entry := ConfigEntry{Name: tt.input}
		result := entry.TerraformAttributeName()
		if result != tt.expected {
			t.Errorf("TerraformAttributeName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTerraformType(t *testing.T) {
	tests := []struct {
		control  string
		expected TerraformType
	}{
		{"string", TerraformTypeString},
		{"password", TerraformTypeString},
		{"textarea", TerraformTypeString},
		{"json", TerraformTypeString},
		{"datetime", TerraformTypeString},
		{"number", TerraformTypeInt64},
		{"boolean", TerraformTypeBool},
		{"toggle", TerraformTypeBool},
		{"one-select", TerraformTypeString},
		{"multi-select", TerraformTypeList},
		{"slider", TerraformTypeInt64},
		{"unknown", TerraformTypeString}, // Default case
		{"", TerraformTypeString},        // Empty string case
	}

	for _, tt := range tests {
		entry := ConfigEntry{Value: ValueObject{Control: tt.control}}
		result := entry.TerraformType()
		if result != tt.expected {
			t.Errorf("TerraformType() for control %q = %q, want %q", tt.control, result, tt.expected)
		}
	}
}

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		name     string
		entry    ConfigEntry
		expected bool
	}{
		{
			name:     "encrypt true",
			entry:    ConfigEntry{Encrypt: true, Value: ValueObject{Control: "string"}},
			expected: true,
		},
		{
			name:     "password control",
			entry:    ConfigEntry{Encrypt: false, Value: ValueObject{Control: "password"}},
			expected: true,
		},
		{
			name:     "both encrypt and password",
			entry:    ConfigEntry{Encrypt: true, Value: ValueObject{Control: "password"}},
			expected: true,
		},
		{
			name:     "not sensitive",
			entry:    ConfigEntry{Encrypt: false, Value: ValueObject{Control: "string"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.IsSensitive()
			if result != tt.expected {
				t.Errorf("IsSensitive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetDefault(t *testing.T) {
	tests := []struct {
		name          string
		defaultValue  any
		expectedStr   string
		expectedInt   int64
		expectedBool  bool
		hasDefault    bool
	}{
		{
			name:         "string default",
			defaultValue: "5432",
			expectedStr:  "5432",
			hasDefault:   true,
		},
		{
			name:         "int default",
			defaultValue: float64(10), // JSON numbers are float64
			expectedStr:  "10",
			expectedInt:  10,
			hasDefault:   true,
		},
		{
			name:         "bool default true",
			defaultValue: true,
			expectedStr:  "true",
			expectedBool: true,
			hasDefault:   true,
		},
		{
			name:         "bool default false",
			defaultValue: false,
			expectedStr:  "false",
			expectedBool: false,
			hasDefault:   true,
		},
		{
			name:         "nil default",
			defaultValue: nil,
			expectedStr:  "",
			hasDefault:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ConfigEntry{Value: ValueObject{Default: tt.defaultValue}}

			if entry.HasDefault() != tt.hasDefault {
				t.Errorf("HasDefault() = %v, want %v", entry.HasDefault(), tt.hasDefault)
			}

			if entry.GetDefaultString() != tt.expectedStr {
				t.Errorf("GetDefaultString() = %q, want %q", entry.GetDefaultString(), tt.expectedStr)
			}

			if entry.GetDefaultInt64() != tt.expectedInt {
				t.Errorf("GetDefaultInt64() = %d, want %d", entry.GetDefaultInt64(), tt.expectedInt)
			}

			if entry.GetDefaultBool() != tt.expectedBool {
				t.Errorf("GetDefaultBool() = %v, want %v", entry.GetDefaultBool(), tt.expectedBool)
			}
		})
	}
}
