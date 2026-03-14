package auth

import (
	"context"
	"testing"
)

func TestStaticValidatorValidToken(t *testing.T) {
	v, err := NewStaticValidator(map[string]string{
		"supersecret": "service-a",
	})
	if err != nil {
		t.Fatalf("NewStaticValidator: %v", err)
	}

	actor, err := v.Validate(context.Background(), "supersecret")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if actor != "service-a" {
		t.Fatalf("want actor %q, got %q", "service-a", actor)
	}
}

func TestStaticValidatorInvalidToken(t *testing.T) {
	v, err := NewStaticValidator(map[string]string{
		"correct-key": "svc",
	})
	if err != nil {
		t.Fatalf("NewStaticValidator: %v", err)
	}
	if _, err := v.Validate(context.Background(), "wrong-key"); err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestStaticValidatorMultipleKeys(t *testing.T) {
	v, err := NewStaticValidator(map[string]string{
		"key-a": "actor-a",
		"key-b": "actor-b",
	})
	if err != nil {
		t.Fatalf("NewStaticValidator: %v", err)
	}

	actorA, err := v.Validate(context.Background(), "key-a")
	if err != nil {
		t.Fatalf("Validate key-a: %v", err)
	}
	if actorA != "actor-a" {
		t.Fatalf("want actor-a, got %q", actorA)
	}

	actorB, err := v.Validate(context.Background(), "key-b")
	if err != nil {
		t.Fatalf("Validate key-b: %v", err)
	}
	if actorB != "actor-b" {
		t.Fatalf("want actor-b, got %q", actorB)
	}
}

func TestStaticValidatorEmptyKeys(t *testing.T) {
	if _, err := NewStaticValidator(map[string]string{}); err == nil {
		t.Fatal("expected error for empty key map, got nil")
	}
}

// TestStaticValidatorTimingConstancy is a smoke test ensuring tokens with
// similar prefixes are not short-circuited. It cannot prove constant-time
// behaviour, but at least verifies correctness.
func TestStaticValidatorTimingConstancy(t *testing.T) {
	v, _ := NewStaticValidator(map[string]string{"abc": "svc"})
	if _, err := v.Validate(context.Background(), "abc-extra"); err == nil {
		t.Fatal("expected rejection for near-match token")
	}
}
