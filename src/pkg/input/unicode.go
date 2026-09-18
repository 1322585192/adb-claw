package input

import (
	"crypto/md5"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/llm-net/adb-claw/pkg/adb"
)

//go:embed classes.dex
var inputDEX []byte

const inputDEXPath = "/data/local/tmp/adbclaw-input.dex"

// EnsureUnicodeDEX pushes the built-in clipboard helper when the device copy
// is missing or differs. The helper runs through app_process and is not an APK.
func EnsureUnicodeDEX(cmd adb.Commander) error {
	if len(inputDEX) < 4 || string(inputDEX[:4]) != "dex\n" {
		return fmt.Errorf("embedded Unicode input DEX is invalid")
	}

	localMD5 := fmt.Sprintf("%x", md5.Sum(inputDEX))
	result, err := cmd.Shell("md5sum", inputDEXPath)
	if err == nil && result.ExitCode == 0 {
		fields := strings.Fields(strings.TrimSpace(result.Stdout))
		if len(fields) > 0 && fields[0] == localMD5 {
			return nil
		}
	}

	tmpFile := filepath.Join(os.TempDir(), "adbclaw-input.dex")
	if err := os.WriteFile(tmpFile, inputDEX, 0o644); err != nil {
		return fmt.Errorf("write Unicode input DEX: %w", err)
	}
	defer os.Remove(tmpFile)

	result, err = cmd.RawCommand("push", tmpFile, inputDEXPath)
	if err != nil {
		return fmt.Errorf("push Unicode input DEX: %w", err)
	}
	if result.ExitCode != 0 {
		message := strings.TrimSpace(result.Stderr + result.Stdout)
		if message == "" {
			message = fmt.Sprintf("adb push returned exit code %d", result.ExitCode)
		}
		return fmt.Errorf("push Unicode input DEX: %s", message)
	}
	return nil
}

// typeUnicode writes text into the focused field through the embedded
// app_process helper. It never installs an APK, changes the IME, or pastes
// into a system window.
func typeUnicode(cmd adb.Commander, text string) error {
	if err := EnsureUnicodeDEX(cmd); err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	result, err := cmd.Shell(
		"CLASSPATH="+inputDEXPath,
		"app_process", "/", "ADBClawInput",
		"--base64", encoded,
	)
	if err != nil {
		return fmt.Errorf("set Unicode text: %w", err)
	}
	if result.ExitCode != 0 || !strings.Contains(result.Stdout, "OK") {
		message := strings.TrimSpace(result.Stderr + result.Stdout)
		if message == "" {
			message = fmt.Sprintf("helper returned exit code %d", result.ExitCode)
		}
		return fmt.Errorf("set Unicode text: %s", message)
	}
	return nil
}
