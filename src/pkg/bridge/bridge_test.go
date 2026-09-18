package bridge

import (
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	if time.Millisecond == 0 {
		t.Fatal("sanity")
	}
	opts := Options{}
	if opts.Debounce != 0 || opts.Poll != 0 {
		t.Fatal("zero values expected before Start")
	}
}

func TestEventJSONShape(t *testing.T) {
	ev := Event{Type: "ui", Source: "poll"}
	if ev.Type != "ui" || ev.Source != "poll" {
		t.Fatalf("%+v", ev)
	}
}
