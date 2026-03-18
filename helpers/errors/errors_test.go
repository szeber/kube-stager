package errors

import (
	"fmt"
	"strings"
	"testing"
)

type mockEnvironmentConfig struct {
	siteName    string
	serviceName string
	environment string
}

func (m mockEnvironmentConfig) GetSiteName() string    { return m.siteName }
func (m mockEnvironmentConfig) GetServiceName() string { return m.serviceName }
func (m mockEnvironmentConfig) GetEnvironment() string { return m.environment }

func TestDatabaseCreationError_Error(t *testing.T) {
	envConfig := mockEnvironmentConfig{siteName: "mysite", serviceName: "myservice", environment: "prod"}

	t.Run("with reason", func(t *testing.T) {
		err := DatabaseCreationError{
			DatabaseType:      DatabaseTypeMysql,
			EnvironmentConfig: envConfig,
			Reason:            "connection refused",
		}
		got := err.Error()
		if !strings.Contains(got, "Mysql") || !strings.Contains(got, "mysite") ||
			!strings.Contains(got, "myservice") || !strings.Contains(got, "prod") ||
			!strings.Contains(got, "connection refused") {
			t.Errorf("unexpected error message: %s", got)
		}
	})

	t.Run("without reason", func(t *testing.T) {
		err := DatabaseCreationError{
			DatabaseType:      DatabaseTypeMongo,
			EnvironmentConfig: envConfig,
		}
		got := err.Error()
		if strings.Contains(got, "Reason") {
			t.Errorf("should not contain Reason when empty: %s", got)
		}
	})
}

func TestDatabaseCreationError_IsFinal(t *testing.T) {
	err := DatabaseCreationError{}
	if !err.IsFinal() {
		t.Error("DatabaseCreationError.IsFinal() should return true")
	}
}

func TestDatabaseInitError_Error(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		err := DatabaseInitError{SiteName: "site1", ServiceName: "svc1", Reason: "timeout"}
		got := err.Error()
		if !strings.Contains(got, "site1") || !strings.Contains(got, "svc1") || !strings.Contains(got, "timeout") {
			t.Errorf("unexpected error message: %s", got)
		}
	})

	t.Run("without reason", func(t *testing.T) {
		err := DatabaseInitError{SiteName: "site1", ServiceName: "svc1"}
		got := err.Error()
		if strings.Contains(got, "Reason") {
			t.Errorf("should not contain Reason when empty: %s", got)
		}
	})
}

func TestDatabaseInitError_IsFinal(t *testing.T) {
	err := DatabaseInitError{}
	if !err.IsFinal() {
		t.Error("DatabaseInitError.IsFinal() should return true")
	}
}

func TestDatabaseMigrationError_Error(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		err := DatabaseMigrationError{SiteName: "site1", ServiceName: "svc1", Reason: "schema conflict"}
		got := err.Error()
		if !strings.Contains(got, "site1") || !strings.Contains(got, "svc1") || !strings.Contains(got, "schema conflict") {
			t.Errorf("unexpected error message: %s", got)
		}
	})

	t.Run("without reason", func(t *testing.T) {
		err := DatabaseMigrationError{SiteName: "site1", ServiceName: "svc1"}
		if strings.Contains(err.Error(), "Reason") {
			t.Errorf("should not contain Reason when empty: %s", err.Error())
		}
	})
}

func TestDatabaseMigrationError_IsFinal(t *testing.T) {
	err := DatabaseMigrationError{}
	if !err.IsFinal() {
		t.Error("DatabaseMigrationError.IsFinal() should return true")
	}
}

func TestUnresolvedTemplatesError_Error(t *testing.T) {
	t.Run("without key", func(t *testing.T) {
		err := UnresolvedTemplatesError{
			UnresolvedTemplateVariables: []string{"${foo}", "${bar}"},
			AvailableTemplateVariables:  []string{"baz", "qux"},
			EntityType:                  "pod spec",
		}
		got := err.Error()
		if !strings.Contains(got, "pod spec") || !strings.Contains(got, "${foo}") {
			t.Errorf("unexpected error message: %s", got)
		}
		if strings.Contains(got, "at key") {
			t.Errorf("should not contain 'at key' when Key is empty: %s", got)
		}
	})

	t.Run("with key", func(t *testing.T) {
		err := UnresolvedTemplatesError{
			UnresolvedTemplateVariables: []string{"${foo}"},
			AvailableTemplateVariables:  []string{"baz"},
			EntityType:                  "configmap",
			Key:                         "data.config",
		}
		got := err.Error()
		if !strings.Contains(got, "at key data.config") {
			t.Errorf("should contain key info: %s", got)
		}
	})
}

func TestUnresolvedTemplatesError_IsFinal(t *testing.T) {
	err := UnresolvedTemplatesError{}
	if !err.IsFinal() {
		t.Error("UnresolvedTemplatesError.IsFinal() should return true")
	}
}

func TestIsControllerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"DatabaseCreationError", DatabaseCreationError{EnvironmentConfig: mockEnvironmentConfig{}}, true},
		{"UnresolvedTemplatesError", UnresolvedTemplatesError{}, true},
		{"DatabaseInitError", DatabaseInitError{}, true},
		{"DatabaseMigrationError", DatabaseMigrationError{}, true},
		{"plain error", fmt.Errorf("some error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsControllerError(tt.err); got != tt.expected {
				t.Errorf("IsControllerError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
