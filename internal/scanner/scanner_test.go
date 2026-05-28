package scanner

import (
	"reflect"
	"testing"
)

func TestResolveCIDTargets_DefaultGuest(t *testing.T) {
	got, err := resolveCIDTargets("", modeGuest, 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []uint32{2, 123}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestResolveCIDTargets_DefaultGuestFallbackAll(t *testing.T) {
	got, err := resolveCIDTargets("", modeGuest, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != int(cidAllMax-cidAllMin+1) {
		t.Fatalf("expected full cid range, got %d", len(got))
	}
}

func TestResolveCIDTargets_CustomInput(t *testing.T) {
	got, err := resolveCIDTargets("3,5-6", modeGuest, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []uint32{3, 5, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestResolvePortTargetsDefaultAll(t *testing.T) {
	got, err := resolvePortTargets("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != int(portAllMax-portAllMin+1) {
		t.Fatalf("expected full port range, got=%d", len(got))
	}
}
