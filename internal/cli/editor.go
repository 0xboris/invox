package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"invox/internal/invoice"
)

var openTextFile = defaultOpenTextFile
var openDocument = defaultOpenDocument
var cleanupOpenedDocument = defaultCleanupOpenedDocument
var preferNativeMailCompose = runtime.GOOS == "darwin"
var openNativeEmailDraft = defaultOpenNativeEmailDraft

func defaultOpenTextFile(path string) error {
	cmd := shellEditorCommand(path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultOpenDocument(path string) error {
	cmd := defaultOpenDocumentCommand(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultOpenNativeEmailDraft(message invoice.EmailMessage) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("native mail compose is unsupported on %s", runtime.GOOS)
	}

	args := []string{
		"-e", `on run argv`,
		"-e", `set recipientAddress to item 1 of argv`,
		"-e", `set messageSubject to item 2 of argv`,
		"-e", `set messageBody to item 3 of argv`,
		"-e", `set attachmentPath to item 4 of argv`,
		"-e", `set senderAddress to item 5 of argv`,
		"-e", `tell application "Mail"`,
		"-e", `activate`,
		"-e", `set draftMessage to make new outgoing message with properties {visible:true, subject:messageSubject, content:messageBody}`,
		"-e", `tell draftMessage`,
		"-e", `make new to recipient at end of to recipients with properties {address:recipientAddress}`,
		"-e", `if senderAddress is not "" then`,
		"-e", `try`,
		"-e", `set sender to senderAddress`,
		"-e", `end try`,
		"-e", `end if`,
		"-e", `delay 0.2`,
		"-e", `make new attachment with properties {file name:(POSIX file attachmentPath)} at after the last paragraph`,
		"-e", `set visible to true`,
		"-e", `end tell`,
		"-e", `end tell`,
		"-e", `end run`,
		"--",
		message.Recipient,
		message.Subject,
		message.Body,
		message.AttachmentPath,
		message.SenderAddress,
	}

	cmd := exec.Command("osascript", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultCleanupOpenedDocument(path string) error {
	const delay = 5 * time.Second

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command(
			"cmd",
			"/c",
			fmt.Sprintf(
				`start "" /b cmd /c "ping -n %d 127.0.0.1 >nul && del /f /q %q"`,
				int(delay/time.Second)+1,
				path,
			),
		)
		return cmd.Run()
	default:
		cmd := exec.Command(
			"/bin/sh",
			"-c",
			fmt.Sprintf(`(sleep %d; rm -f "$1") >/dev/null 2>&1 &`, int(delay/time.Second)),
			"invox",
			path,
		)
		return cmd.Run()
	}
}

func defaultOpenDocumentCommand(path string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path)
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path)
	default:
		return exec.Command("xdg-open", path)
	}
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
