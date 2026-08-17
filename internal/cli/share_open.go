package cli

import (
	"context"
	"os/exec"
	"runtime"
)

// openInBrowser hands path to whatever the desktop uses to open a file. It is the only
// platform-specific call assaio makes, and it is deliberately best-effort: the command
// has already written the file and printed its path, so a machine with no desktop (a
// container, a remote shell) loses nothing by this failing.
//
// This opens a local file. Nothing here reaches the network -- the preview page makes no
// request either, and posting is the person's own action in their own browser.
func openInBrowser(ctx context.Context, path string) error {
	name, args := openCommand(path)
	if name == "" {
		return nil
	}
	return exec.CommandContext(ctx, name, args...).Start() //nolint:gosec // name is one of three literals below and path is a file this process just wrote.
}

// openCommand names the opener for the running OS. Windows goes through rundll32 rather
// than "cmd /c start" because start treats its first quoted argument as a window title,
// which silently opens nothing when a path contains a space.
func openCommand(path string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{path}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", path}
	case "linux", "freebsd", "openbsd", "netbsd":
		return "xdg-open", []string{path}
	default:
		return "", nil
	}
}
