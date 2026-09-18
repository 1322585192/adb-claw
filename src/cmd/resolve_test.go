package cmd

import "testing"

func TestResolveErrorCode(t *testing.T) {
	if got := resolveErrorCode(errString("no cached UI snapshot; run observe first")); got != "STALE_STATE" {
		t.Errorf("got %s", got)
	}
	if got := resolveErrorCode(errString("ui tree dump failed: boom")); got != "UI_DUMP_FAILED" {
		t.Errorf("got %s", got)
	}
	if got := resolveErrorCode(errString("index 3 out of range")); got != "ELEMENT_NOT_FOUND" {
		t.Errorf("got %s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
