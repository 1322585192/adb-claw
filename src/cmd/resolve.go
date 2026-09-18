package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/observe"
)

func resolveElementByIndex(index int) (*observe.Element, error) {
	tree, err := loadActionTree()
	if err != nil {
		return nil, err
	}
	return tree.FindByIndex(index)
}

func resolveElementByID(id string) (*observe.Element, error) {
	tree, err := loadActionTree()
	if err != nil {
		return nil, err
	}
	results := tree.FindByID(id)
	if len(results) == 0 {
		return nil, fmt.Errorf("no element found with id '%s'", id)
	}
	return &results[0], nil
}

func resolveElementByText(text string) (*observe.Element, error) {
	tree, err := loadActionTree()
	if err != nil {
		return nil, err
	}
	results := tree.FindByText(text)
	if len(results) == 0 {
		return nil, fmt.Errorf("no element found with text '%s'", text)
	}
	return &results[0], nil
}

func loadActionTree() (*observe.UITree, error) {
	if refreshElements {
		tree, err := observe.DumpUITreeOpts(client, observe.DumpOptions{Save: true})
		if err != nil {
			return nil, fmt.Errorf("ui tree dump failed: %w", err)
		}
		return tree, nil
	}

	snap, err := observe.LoadSnapshot(client.Serial, observe.DefaultSnapshotMaxAge)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	age := time.Since(snap.CapturedAt).Round(time.Millisecond)
	writer.Verbose("using cached snapshot %s age=%s", snap.StateID, age)
	return snap.Tree, nil
}

func resolveErrorCode(err error) string {
	if err == nil {
		return "ELEMENT_NOT_FOUND"
	}
	msg := err.Error()
	if containsAny(msg, "cached UI snapshot", "stale state_id", "expired") {
		return "STALE_STATE"
	}
	if containsAny(msg, "ui tree dump failed") {
		return "UI_DUMP_FAILED"
	}
	return "ELEMENT_NOT_FOUND"
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func elementInfo(el *observe.Element) map[string]interface{} {
	info := map[string]interface{}{
		"index": el.Index,
	}
	if el.Handle != "" {
		info["handle"] = el.Handle
	}
	if el.Text != "" {
		info["text"] = el.Text
	}
	if el.ResourceID != "" {
		info["resource_id"] = el.ResourceID
	}
	if el.ContentDesc != "" {
		info["content_desc"] = el.ContentDesc
	}
	return info
}
