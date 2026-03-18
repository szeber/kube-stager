package v1

import (
	"context"
	"strings"
	"testing"

	"github.com/szeber/kube-stager/helpers/annotations"
)

func TestStagingSiteDefaulter_Default(t *testing.T) {
	d := &StagingSiteDefaulter{}

	t.Run("empty fields get defaults", func(t *testing.T) {
		site := &StagingSite{}
		site.Name = "mysite"
		err := d.Default(context.Background(), site)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if site.Spec.DomainPrefix != "mysite" {
			t.Errorf("DomainPrefix = %q, want %q", site.Spec.DomainPrefix, "mysite")
		}
		if site.Spec.DbName == "" {
			t.Error("DbName should be set")
		}
		if site.Spec.Username == "" {
			t.Error("Username should be set")
		}
		if site.Spec.Password == "" {
			t.Error("Password should be generated")
		}
		if site.Spec.DisableAfter.Days != 2 {
			t.Errorf("DisableAfter.Days = %d, want 2", site.Spec.DisableAfter.Days)
		}
		if site.Spec.DeleteAfter.Days != 7 {
			t.Errorf("DeleteAfter.Days = %d, want 7", site.Spec.DeleteAfter.Days)
		}
	})

	t.Run("annotation set", func(t *testing.T) {
		site := &StagingSite{}
		site.Name = "mysite"
		site.Spec.DomainPrefix = "preset"
		site.Spec.DbName = "presetdb"
		site.Spec.Username = "presetuser"
		site.Spec.Password = "presetpass"
		site.Spec.DisableAfter = TimeInterval{Never: true}
		site.Spec.DeleteAfter = TimeInterval{Never: true}
		if err := d.Default(context.Background(), site); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := site.ObjectMeta.Annotations[annotations.StagingSiteLastSpecChangeAt]; !ok {
			t.Error("LastSpecChangeAt annotation should be set")
		}
	})

	t.Run("long name truncated for DomainPrefix", func(t *testing.T) {
		site := &StagingSite{}
		site.Name = strings.Repeat("a", 100)
		site.Spec.DisableAfter = TimeInterval{Never: true}
		site.Spec.DeleteAfter = TimeInterval{Never: true}
		if err := d.Default(context.Background(), site); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(site.Spec.DomainPrefix) > 63 {
			t.Errorf("DomainPrefix length = %d, want <= 63", len(site.Spec.DomainPrefix))
		}
	})

	t.Run("preset values not overridden", func(t *testing.T) {
		site := &StagingSite{}
		site.Name = "mysite"
		site.Spec.DomainPrefix = "custom"
		site.Spec.DbName = "customdb"
		site.Spec.Username = "customuser"
		site.Spec.Password = "custompass"
		site.Spec.DisableAfter = TimeInterval{Never: true}
		site.Spec.DeleteAfter = TimeInterval{Never: true}
		if err := d.Default(context.Background(), site); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if site.Spec.DomainPrefix != "custom" {
			t.Errorf("DomainPrefix = %q, want %q", site.Spec.DomainPrefix, "custom")
		}
		if site.Spec.DbName != "customdb" {
			t.Errorf("DbName = %q, want %q", site.Spec.DbName, "customdb")
		}
		if site.Spec.Username != "customuser" {
			t.Errorf("Username = %q, want %q", site.Spec.Username, "customuser")
		}
		if site.Spec.Password != "custompass" {
			t.Errorf("Password = %q, want %q", site.Spec.Password, "custompass")
		}
	})
}
