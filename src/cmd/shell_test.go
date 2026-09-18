package cmd

import "testing"

func TestForbiddenShellReason(t *testing.T) {
	tests := []struct {
		name    string
		command string
		block   bool
	}{
		{"ui dump", "uiautomator " + "dump /sdcard/ui.xml", true},
		{"ui dump in shell", "sh -c 'uiautomator " + "dump /sdcard/ui.xml'", true},
		{"window layout", "dumpsys window displays", true},
		{"activity top", "dumpsys activity top", true},
		{"accessibility", "dumpsys accessibility", true},
		{"clipboard binder", "service call clipboard 2 s16 text", true},
		{"set ime", "ime set com.android.adbkeyboard/.AdbIME", true},
		{"enable ime", "ime enable com.android.adbkeyboard/.AdbIME", true},
		{"secure ime", "settings put secure default_input_method bad/.IME", true},
		{"get property", "getprop ro.build.version.release", false},
		{"list files", "ls /sdcard/", false},
		{"package metadata", "dumpsys package com.example", false},
		{"ime list read only", "ime list -s", false},
		{"battery diagnostics", "dumpsys battery", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := forbiddenShellReason(tt.command)
			if got := reason != ""; got != tt.block {
				t.Fatalf("forbiddenShellReason(%q) = %q, blocked=%v; want blocked=%v",
					tt.command, reason, got, tt.block)
			}
		})
	}
}
