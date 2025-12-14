// amp-sandboxed-v2 - Reverse sandbox with blocklist approach
//
// Architecture:
//   - Default: ALLOW writes everywhere
//   - Blocklist: Protect sensitive paths (.git, ~/.ssh, ~/.aws, etc.)
//   - This acts as a kernel-level "hard stop" for critical areas
//
// Build: go build -o amp-sandboxed-v2 amp-sandboxed-v2.go
// Usage: ./amp-sandboxed-v2 [amp args...]

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const basePolicy = `(version 1)

; Reverse sandbox: allow all, blocklist sensitive paths only
(allow default)
(allow file-write*)
`

type BlockedPath struct {
	Path        string
	Description string
	IsRegex     bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[0;31m[sandbox]\033[0m %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ampBin, err := findAmp()
	if err != nil {
		return err
	}

	home := os.Getenv("HOME")
	blockedPaths := loadBlockedPaths(home)

	policy := buildReversePolicy(blockedPaths)

	args := []string{"-p", policy, "--", ampBin}
	args = append(args, os.Args[1:]...)

	if os.Getenv("DEBUG_SANDBOX") != "" {
		fmt.Println("=== POLICY ===")
		fmt.Println(policy)
		fmt.Println("=== BLOCKED PATHS ===")
		for _, bp := range blockedPaths {
			fmt.Printf("  %s (%s)\n", bp.Path, bp.Description)
		}
		return nil
	}

	return syscall.Exec("/usr/bin/sandbox-exec", append([]string{"sandbox-exec"}, args...), os.Environ())
}

func buildReversePolicy(blocked []BlockedPath) string {
	var sb strings.Builder
	sb.WriteString(basePolicy)

	sb.WriteString("\n; Blocked paths (kernel-level protection)\n")
	for _, bp := range blocked {
		if bp.Path == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("; %s\n", bp.Description))
		sb.WriteString(fmt.Sprintf("(deny file-write* (subpath %q))\n", bp.Path))
	}

	return sb.String()
}

func loadBlockedPaths(home string) []BlockedPath {
	configPath := filepath.Join(home, ".config/amp-permissions/sandbox-blocked-paths.txt")

	var paths []BlockedPath

	data, err := os.ReadFile(configPath)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Expand $HOME
			expanded := strings.ReplaceAll(line, "$HOME", home)
			paths = append(paths, BlockedPath{
				Path:        expanded,
				Description: "from config",
			})
		}
	}

	// Fallback defaults if config empty/missing
	if len(paths) == 0 {
		paths = []BlockedPath{
			{Path: filepath.Join(home, ".ssh"), Description: "SSH keys"},
			{Path: filepath.Join(home, ".gnupg"), Description: "GPG keys"},
			{Path: filepath.Join(home, ".aws"), Description: "AWS credentials"},
			{Path: filepath.Join(home, ".kube"), Description: "Kubernetes config"},
			{Path: filepath.Join(home, "Library/Keychains"), Description: "macOS Keychains"},
			{Path: "/etc", Description: "System config"},
			{Path: "/System", Description: "macOS System"},
			{Path: "/Library", Description: "System Library"},
		}
	}

	return paths
}

func findAmp() (string, error) {
	if path, err := exec.LookPath("amp"); err == nil {
		return path, nil
	}

	home := os.Getenv("HOME")
	locations := []string{
		filepath.Join(home, ".bun/bin/amp"),
		"/usr/local/bin/amp",
		"/opt/homebrew/bin/amp",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", fmt.Errorf("amp not found in PATH or common locations")
}

func getDarwinCacheDir() string {
	out, err := exec.Command("getconf", "DARWIN_USER_CACHE_DIR").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return filepath.Join(os.Getenv("HOME"), "Library/Caches")
}
