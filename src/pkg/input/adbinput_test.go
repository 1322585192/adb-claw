package input

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/llm-net/adb-claw/pkg/adb"
)

type typeMock struct {
	shells [][]string
	pushes [][]string
	fail   string
}

func (m *typeMock) Shell(args ...string) (*adb.Result, error) {
	copied := append([]string{}, args...)
	m.shells = append(m.shells, copied)
	joined := strings.Join(args, " ")
	switch {
	case len(args) > 0 && args[0] == "md5sum":
		return &adb.Result{Stdout: fmt.Sprintf("%x  %s\n", md5.Sum(inputDEX), inputDEXPath)}, nil
	case strings.Contains(joined, "ADBClawInput"):
		if m.fail == "helper" {
			return &adb.Result{Stderr: "clipboard denied", ExitCode: 1}, nil
		}
		return &adb.Result{Stdout: "OK SET_TEXT\n"}, nil
	case joined == "input keyevent KEYCODE_PASTE":
		return &adb.Result{Stderr: "unexpected paste", ExitCode: 1}, nil
	default:
		return &adb.Result{}, nil
	}
}

func (m *typeMock) ExecOut(args ...string) ([]byte, error) {
	return nil, fmt.Errorf("unexpected ExecOut: %v", args)
}

func (m *typeMock) RawCommand(args ...string) (*adb.Result, error) {
	m.pushes = append(m.pushes, append([]string{}, args...))
	return &adb.Result{}, nil
}

func TestEscapeForInput(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello%sworld"},
		{"it's", "it\\'s"},
		{"a&b", "a\\&b"},
		{"test(1)", "test\\(1\\)"},
	}

	for _, tt := range tests {
		got := escapeForInput(tt.input)
		if got != tt.want {
			t.Errorf("escapeForInput(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveKeycode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HOME", "KEYCODE_HOME"},
		{"home", "KEYCODE_HOME"},
		{"BACK", "KEYCODE_BACK"},
		{"ENTER", "KEYCODE_ENTER"},
		{"KEYCODE_HOME", "KEYCODE_HOME"},
		{"VOLUME_UP", "KEYCODE_VOLUME_UP"},
		{"SPACE", "KEYCODE_SPACE"},
		{"UNKNOWN_KEY", "KEYCODE_UNKNOWN_KEY"},
		// New aliases added in iteration 2
		{"PASTE", "KEYCODE_PASTE"},
		{"COPY", "KEYCODE_COPY"},
		{"CUT", "KEYCODE_CUT"},
		{"FORWARD_DEL", "KEYCODE_FORWARD_DEL"},
		{"MOVE_HOME", "KEYCODE_MOVE_HOME"},
		{"MOVE_END", "KEYCODE_MOVE_END"},
		{"PAGE_UP", "KEYCODE_PAGE_UP"},
		{"PAGE_DOWN", "KEYCODE_PAGE_DOWN"},
		{"WAKEUP", "KEYCODE_WAKEUP"},
		{"SLEEP", "KEYCODE_SLEEP"},
	}

	for _, tt := range tests {
		got := resolveKeycode(tt.input)
		if got != tt.want {
			t.Errorf("resolveKeycode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTypeTextASCIIUsesADBInput(t *testing.T) {
	cmd := &typeMock{}
	method, err := TypeText(cmd, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if method != TypeMethodADBInput {
		t.Fatalf("method = %q, want %q", method, TypeMethodADBInput)
	}
	if len(cmd.shells) != 1 || strings.Join(cmd.shells[0], " ") != "input text hello%sworld" {
		t.Fatalf("shell calls = %v", cmd.shells)
	}
}

func TestTypeTextUnicodeUsesEmbeddedClipboardHelper(t *testing.T) {
	cmd := &typeMock{}
	text := "王者荣耀"
	method, err := TypeText(cmd, text)
	if err != nil {
		t.Fatal(err)
	}
	if method != TypeMethodUnicodeClipboard {
		t.Fatalf("method = %q, want %q", method, TypeMethodUnicodeClipboard)
	}
	if len(cmd.pushes) != 0 {
		t.Fatalf("matching embedded DEX should not be pushed: %v", cmd.pushes)
	}
	if len(cmd.shells) != 2 {
		t.Fatalf("shell calls = %v", cmd.shells)
	}
	helper := cmd.shells[1]
	if got := helper[len(helper)-1]; got != base64.StdEncoding.EncodeToString([]byte(text)) {
		t.Fatalf("helper payload = %q", got)
	}
	if strings.Contains(fmt.Sprint(cmd.shells), "KEYCODE_PASTE") {
		t.Fatalf("SET_TEXT must not paste into a system window: %v", cmd.shells)
	}
}

func TestTypeTextUnicodeFailureForbidsIMEWorkaround(t *testing.T) {
	cmd := &typeMock{fail: "helper"}
	_, err := TypeText(cmd, "中文")
	if err == nil {
		t.Fatal("expected Unicode helper failure")
	}
	message := err.Error()
	if !strings.Contains(message, "do not install an IME") {
		t.Fatalf("error must close the third-party IME escape route: %q", message)
	}
}

func TestEmbeddedInputDEX(t *testing.T) {
	if len(inputDEX) < 4 || string(inputDEX[:4]) != "dex\n" {
		t.Fatalf("embedded input DEX has invalid magic")
	}
}
