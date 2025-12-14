// amp-sandboxed - Run Amp with macOS sandbox isolation
// Based on OpenAI Codex CLI seatbelt policies
//
// Build: go build -o amp-sandboxed amp-sandboxed.go
// Usage: ./amp-sandboxed [amp args...]

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

; Allow-default approach - only restrict file writes
(allow default)

; Deny file writes by default (will allow specific paths below)
(deny file-write*)
`

// WritableRoot represents a path that can be written to, with optional read-only subpaths
type WritableRoot struct {
	Root             string
	ReadOnlySubpaths []string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[0;31m[sandbox]\033[0m %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Find amp binary
	ampBin, err := findAmp()
	if err != nil {
		return err
	}

	// Get canonical PWD
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	canonicalPwd, err := filepath.EvalSymlinks(pwd)
	if err != nil {
		canonicalPwd = pwd
	}

	// Get TMPDIR
	tmpdir := os.Getenv("TMPDIR")
	if tmpdir == "" {
		tmpdir = "/tmp"
	}
	canonicalTmpdir, err := filepath.EvalSymlinks(tmpdir)
	if err != nil {
		canonicalTmpdir = tmpdir
	}

	// Get Darwin user cache dir
	cacheDir := getDarwinCacheDir()

	// Get home dir for amp config paths
	home := os.Getenv("HOME")

	// Get user's var/folders base path (parent of TMPDIR)
	varFoldersPath := filepath.Dir(filepath.Dir(canonicalTmpdir)) // e.g. /private/var/folders/xx/yyyyyyyy

	// Build writable roots
	// Narrowed to amp-specific paths only
	writableRoots := []WritableRoot{
		{Root: canonicalPwd, ReadOnlySubpaths: nil},
		{Root: canonicalTmpdir, ReadOnlySubpaths: nil},
		{Root: "/private/tmp", ReadOnlySubpaths: nil},
		{Root: "/tmp", ReadOnlySubpaths: nil},
		{Root: "/dev/null", ReadOnlySubpaths: nil},
		{Root: cacheDir, ReadOnlySubpaths: nil},
		{Root: filepath.Join(home, ".amp"), ReadOnlySubpaths: nil},
		{Root: filepath.Join(home, ".config/amp"), ReadOnlySubpaths: nil},
		{Root: filepath.Join(home, ".cache/amp"), ReadOnlySubpaths: nil},
		{Root: filepath.Join(home, ".local/share/amp"), ReadOnlySubpaths: nil},
		{Root: varFoldersPath, ReadOnlySubpaths: nil},                                     // User's var/folders (e.g. /private/var/folders/xx/yy)
		{Root: strings.Replace(varFoldersPath, "/private", "", 1), ReadOnlySubpaths: nil}, // Symlink path (/var/folders/xx/yy)
	}

	// Generate dynamic policy
	policy := buildPolicy(writableRoots)

	// Build sandbox-exec args
	args := []string{"-p", policy, "--", ampBin}
	args = append(args, os.Args[1:]...)

	// Debug: print policy and args
	if os.Getenv("DEBUG_SANDBOX") != "" {
		fmt.Println("=== POLICY ===")
		fmt.Println(policy)
		fmt.Println("=== ARGS ===")
		fmt.Println(args)
		return nil
	}

	// Exec sandbox-exec (replaces current process)
	return syscall.Exec("/usr/bin/sandbox-exec", append([]string{"sandbox-exec"}, args...), os.Environ())
}

func buildPolicy(roots []WritableRoot) string {
	var sb strings.Builder
	sb.WriteString(basePolicy)
	sb.WriteString("\n; Dynamic file-write rules\n")
	sb.WriteString("(allow file-write*\n")

	for _, root := range roots {
		if len(root.ReadOnlySubpaths) == 0 {
			// Simple case: entire subtree is writable
			sb.WriteString(fmt.Sprintf("  (subpath %q)\n", root.Root))
		} else {
			// Complex case: writable except for certain subpaths
			sb.WriteString("  (require-all\n")
			sb.WriteString(fmt.Sprintf("    (subpath %q)\n", root.Root))
			for _, ro := range root.ReadOnlySubpaths {
				sb.WriteString(fmt.Sprintf("    (require-not (subpath %q))\n", ro))
			}
			sb.WriteString("  )\n")
		}
	}

	sb.WriteString(")\n")
	return sb.String()
}

func findAmp() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("amp"); err == nil {
		return path, nil
	}

	// Check common locations
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
	// Try getconf first
	out, err := exec.Command("getconf", "DARWIN_USER_CACHE_DIR").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	// Fallback
	return filepath.Join(os.Getenv("HOME"), "Library/Caches")
}

func logInfo(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\033[0;32m[sandbox]\033[0m "+format+"\n", args...)
}
