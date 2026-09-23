package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const canary = "s3cr3t-canary-value"

func TestSecretNeverFormatsItsValue(t *testing.T) {
	s := New(canary)

	var logBuf bytes.Buffer
	slog.New(slog.NewTextHandler(&logBuf, nil)).Info("msg", "token", s)
	jsonOut, err := json.Marshal(struct{ T Secret }{s})
	if err != nil {
		t.Fatal(err)
	}

	outputs := map[string]string{
		"%v":      fmt.Sprintf("%v", s),
		"%+v":     fmt.Sprintf("%+v", s),
		"%#v":     fmt.Sprintf("%#v", s),
		"%s":      fmt.Sprintf("%s", s),
		"%q":      fmt.Sprintf("%q", s),
		"%x":      fmt.Sprintf("%x", s),
		"struct":  fmt.Sprintf("%+v", struct{ T Secret }{s}),
		"slog":    logBuf.String(),
		"json":    string(jsonOut),
		"Println": fmt.Sprintln(s),
	}
	for name, out := range outputs {
		if strings.Contains(out, canary) {
			t.Errorf("%s leaked the secret: %q", name, out)
		}
	}
}

func TestRevealReturnsValue(t *testing.T) {
	if got := New(canary).Reveal(); got != canary {
		t.Fatalf("Reveal() = %q", got)
	}
}

func TestEmpty(t *testing.T) {
	if !New("").Empty() || New("x").Empty() {
		t.Fatal("Empty() wrong")
	}
}
