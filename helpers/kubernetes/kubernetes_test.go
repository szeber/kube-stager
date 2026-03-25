package kubernetes

import (
	"context"
	"testing"

	"github.com/szeber/kube-stager/internal/testutil"
)

func TestGetServiceConfigsInNamespace(t *testing.T) {
	t.Run("returns all configs", func(t *testing.T) {
		sc1 := testutil.NewTestServiceConfig("svc1", "test-ns", "s1")
		sc2 := testutil.NewTestServiceConfig("svc2", "test-ns", "s2")
		c := testutil.NewFakeClient(sc1, sc2)
		result, err := GetServiceConfigsInNamespace("test-ns", c, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("got %d configs, want 2", len(result))
		}
		if _, ok := result["svc1"]; !ok {
			t.Error("missing svc1")
		}
	})

	t.Run("empty namespace returns empty map", func(t *testing.T) {
		sc := testutil.NewTestServiceConfig("svc1", "other-ns", "s1")
		c := testutil.NewFakeClient(sc)
		result, err := GetServiceConfigsInNamespace("test-ns", c, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("got %d configs, want 0", len(result))
		}
	})
}

func TestGetMysqlEnvironmentsInNamespace(t *testing.T) {
	mc := testutil.NewTestMysqlConfig("mysql1", "test-ns")
	c := testutil.NewFakeClient(mc)
	result, err := GetMysqlEnvironmentsInNamespace("test-ns", c, context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d configs, want 1", len(result))
	}
	if _, ok := result["mysql1"]; !ok {
		t.Error("missing mysql1")
	}
}

func TestGetMongoEnvironmentsInNamespace(t *testing.T) {
	mc := testutil.NewTestMongoConfig("mongo1", "test-ns")
	c := testutil.NewFakeClient(mc)
	result, err := GetMongoEnvironmentsInNamespace("test-ns", c, context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d configs, want 1", len(result))
	}
}

func TestGetRedisEnvironmentsInNamespace(t *testing.T) {
	rc := testutil.NewTestRedisConfig("redis1", "test-ns")
	c := testutil.NewFakeClient(rc)
	result, err := GetRedisEnvironmentsInNamespace("test-ns", c, context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d configs, want 1", len(result))
	}
}
