package app_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hz/skillctl/internal/app"
)

func TestFailureEnvelope(t *testing.T) {
	err := app.WithHint(app.WithPath(app.Fail("validation", "SKILL.md not found"), "/tmp/x"), "create SKILL.md")
	var buf bytes.Buffer
	if werr := app.WriteJSON(&buf, app.Failure(err)); werr != nil {
		t.Fatal(werr)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != false {
		t.Fatalf("ok=%v", got["ok"])
	}
	e := got["error"].(map[string]any)
	if e["kind"] != "validation" || e["message"] != "SKILL.md not found" {
		t.Fatalf("error=%v", e)
	}
}

func TestSuccessEnvelope(t *testing.T) {
	var buf bytes.Buffer
	if err := app.WriteJSON(&buf, app.Success(map[string]any{"path": "/tmp"})); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != true || got["path"] != "/tmp" {
		t.Fatalf("got %v", got)
	}
}
