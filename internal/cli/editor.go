package cli

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var openTextFile = defaultOpenTextFile

func defaultOpenTextFile(path string) error {
	cmd := shellEditorCommand(path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellEditorCommand(path string) *exec.Cmd {
	editor := resolveShellEditor()

	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", editor, path)
		cmd.Env = append(os.Environ(), "INVOX_EDITOR="+editor)
		return cmd
	}

	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-lc", `eval "$INVOX_EDITOR" '"$1"'`, "invox", path)
	cmd.Env = append(os.Environ(), "INVOX_EDITOR="+editor)
	return cmd
}

func resolveShellEditor() string {
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("EDITOR")); editor != "" {
		return editor
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
