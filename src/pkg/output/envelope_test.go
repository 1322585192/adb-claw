package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestWriterSuccess(t *testing.T) {
	var buf bytes.Buffer
	w := &Writer{out: &buf, format: "json"}

	start := time.Now()
	w.Success("test", map[string]string{"foo": "bar"}, start)

	var resp Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.OK {
		t.Error("expected ok=true")
	}
	if resp.Command != "test" {
		t.Errorf("command = %q, want %q", resp.Command, "test")
	}
	if resp.Error != nil {
		t.Error("expected no error")
	}
	if resp.Timestamp == "" {
		t.Error("expected timestamp")
	}
}

func TestWriterSuccessCompact(t *testing.T) {
	var buf bytes.Buffer
	w := &Writer{out: &buf, format: "json"}

	start := time.Now()
	w.SuccessCompact("observe", map[string]string{"foo": "bar"}, start)

	out := buf.String()
	if strings.Contains(out, "\n  ") {
		t.Errorf("compact JSON should not be indented: %s", out)
	}
	var resp Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal compact response: %v", err)
	}
	if !resp.OK || resp.Command != "observe" {
		t.Errorf("unexpected compact response: %+v", resp)
	}
}

func TestWriterFail(t *testing.T) {
	var buf bytes.Buffer
	w := &Writer{out: &buf, format: "json"}

	start := time.Now()
	w.Fail("test", "TEST_ERROR", "something failed", "try again", start)

	var resp Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.OK {
		t.Error("expected ok=false")
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != "TEST_ERROR" {
		t.Errorf("error code = %q, want %q", resp.Error.Code, "TEST_ERROR")
	}
	if resp.Error.Message != "something failed" {
		t.Errorf("error message = %q", resp.Error.Message)
	}
	if resp.Error.Suggestion != "try again" {
		t.Errorf("suggestion = %q", resp.Error.Suggestion)
	}
}
