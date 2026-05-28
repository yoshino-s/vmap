package scanner

import (
	"reflect"
	"strings"
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

func TestNewScanner_InvalidLogLevel(t *testing.T) {
	_, err := New(Options{
		Mode:      modeHost,
		CIDInput:  "2",
		PortInput: "1",
		LogLevel:  "trace",
	})
	if err == nil {
		t.Fatal("expected invalid log level error")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildDetectPayloads_Default(t *testing.T) {
	payloads, err := buildDetectPayloads(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payloads) != 6 {
		t.Fatalf("expected 6 payloads, got=%d", len(payloads))
	}
}

func TestBuildDetectPayloads_Custom(t *testing.T) {
	payloads, err := buildDetectPayloads([]string{"hello", "\\n", "empty"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := [][]byte{[]byte("hello"), []byte("\n"), []byte{}}
	if !reflect.DeepEqual(payloads, want) {
		t.Fatalf("payload mismatch: got=%q want=%q", payloads, want)
	}
}
