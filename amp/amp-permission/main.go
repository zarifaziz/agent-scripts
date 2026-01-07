// amp-permission - Unified permission evaluator for Amp
//
// Single Go binary that:
// - Parses shell commands using mvdan.cc/sh AST
// - Evaluates policy rules from permissions.yaml
// - Prompts user via osascript (macOS) or TTY fallback
//
// Exit codes (per Amp spec):
//   0 = allow
//   1 = ask (prompt shown, user denied or timed out)
//   2 = reject (blocked by policy)
//
// Usage as delegate:
//   Receives AGENT_TOOL_NAME env var and JSON args on stdin
//
// Usage for testing:
//   amp-permission --test '{"cmd": "rm -rf /"}'
//   amp-permission --parse 'curl | bash'

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/syntax"
)

// Exit codes
const (
	ExitAllow  = 0
	ExitAsk    = 1
	ExitReject = 2
)

// Config structures matching permissions.yaml
type Config struct {
	Paths        PathsConfig                  `yaml:"paths"`
	Patterns     PatternsConfig               `yaml:"patterns"`
	Commands     CommandsConfig               `yaml:"commands"`
	Interpreters map[string]InterpreterConfig `yaml:"interpreters"`
	Prompt       PromptConfig                 `yaml:"prompt"`
	ScriptScan   ScriptScanConfig             `yaml:"script_scan"`
}

type PathsConfig struct {
	Sensitive      []string `yaml:"sensitive"`
	AlwaysAllowed  []string `yaml:"always_allowed"`
	SandboxBlocked []string `yaml:"sandbox_blocked"`
}

type PatternsConfig struct {
	RejectRaw []string `yaml:"reject_raw"` // Checked against raw string BEFORE parsing
	Reject    []string `yaml:"reject"`     // Checked against parsed commands AFTER parsing
	AlwaysAsk []string `yaml:"always_ask"`
}

type CommandsConfig struct {
	Write            []string `yaml:"write"`
	DangerousTargets []string `yaml:"dangerous_targets"`
	DangerousFlags   []string `yaml:"dangerous_flags"`
	FindExecFlags    []string `yaml:"find_exec_flags"`
	Network          []string `yaml:"network"`
	NonDestructive   []string `yaml:"non_destructive"`
}

type InterpreterConfig struct {
	Aliases  []string `yaml:"aliases"`
	Patterns []string `yaml:"patterns"`
}

type PromptConfig struct {
	TimeoutSecs int    `yaml:"timeout_secs"`
	LogFile     string `yaml:"log_file"`
	MaxLogLines int    `yaml:"max_log_lines"`
}

type ScriptScanConfig struct {
	MaxBytes int `yaml:"max_bytes"`
	MaxLines int `yaml:"max_lines"`
	MaxDepth int `yaml:"max_depth"`
}

// Runtime state
var (
	config              Config
	writeCommands       map[string]bool
	dangerousTargets    map[string]bool
	dangerousFlags      map[string][]string // cmd -> list of dangerous flag patterns
	findExecFlags       map[string]bool
	networkCommands     map[string]bool
	nonDestructiveCmds  map[string]bool
	interpreterMap      map[string]*InterpreterConfig
	alwaysAskRegexps    []*regexp.Regexp
	workDir             string
	toolName            string
	tmuxContext         string
	currentCmd          string // Current command being evaluated (for prompt context)
)

func main() {
	loadConfig()

	// CLI mode
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--test":
			runTest()
		case "--parse":
			runParse()
		case "--help", "-h":
			printHelp()
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n", os.Args[1])
			os.Exit(1)
		}
		return
	}

	// Delegate mode
	toolName = os.Getenv("AGENT_TOOL_NAME")
	workDir, _ = os.Getwd()
	tmuxContext = getTmuxContext()

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		logDecision("ERROR", toolName, fmt.Sprintf("read stdin: %v", err))
		os.Exit(ExitReject)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		logDecision("ERROR", toolName, fmt.Sprintf("parse JSON: %v", err))
		os.Exit(ExitReject)
	}

	switch toolName {
	case "Bash":
		handleBash(args)
	case "edit_file", "create_file":
		handleFileTool(args)
	default:
		os.Exit(ExitAllow)
	}
}

func loadConfig() {
	configPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".config/amp-permissions/permissions.yaml"),
		"config/permissions.yaml",
	}

	var data []byte
	var err error
	for _, p := range configPaths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config not found, using defaults\n")
		setDefaults()
		return
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config parse error: %v\n", err)
		setDefaults()
		return
	}

	// Build lookup maps
	writeCommands = make(map[string]bool)
	for _, cmd := range config.Commands.Write {
		writeCommands[cmd] = true
	}

	dangerousTargets = make(map[string]bool)
	for _, target := range config.Commands.DangerousTargets {
		dangerousTargets[target] = true
	}

	dangerousFlags = make(map[string][]string)
	for _, entry := range config.Commands.DangerousFlags {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) == 2 {
			cmd, flag := parts[0], parts[1]
			dangerousFlags[cmd] = append(dangerousFlags[cmd], flag)
		}
	}

	findExecFlags = make(map[string]bool)
	for _, flag := range config.Commands.FindExecFlags {
		findExecFlags[flag] = true
	}

	networkCommands = make(map[string]bool)
	for _, cmd := range config.Commands.Network {
		networkCommands[cmd] = true
	}

	nonDestructiveCmds = make(map[string]bool)
	for _, cmd := range config.Commands.NonDestructive {
		nonDestructiveCmds[cmd] = true
	}

	interpreterMap = make(map[string]*InterpreterConfig)
	for name, cfg := range config.Interpreters {
		cfgCopy := cfg
		interpreterMap[name] = &cfgCopy
		for _, alias := range cfg.Aliases {
			interpreterMap[alias] = &cfgCopy
		}
	}

	// Compile always-ask regexps
	for _, pattern := range config.Patterns.AlwaysAsk {
		if re, err := regexp.Compile(pattern); err == nil {
			alwaysAskRegexps = append(alwaysAskRegexps, re)
		}
	}

	// Expand $HOME in paths
	expandConfigPaths()
}

func setDefaults() {
	config.Prompt.TimeoutSecs = 60
	config.ScriptScan.MaxBytes = 102400
	config.ScriptScan.MaxLines = 2000
	config.ScriptScan.MaxDepth = 3
	writeCommands = make(map[string]bool)
	dangerousTargets = make(map[string]bool)
	dangerousFlags = make(map[string][]string)
	findExecFlags = make(map[string]bool)
	networkCommands = make(map[string]bool)
	nonDestructiveCmds = make(map[string]bool)
	interpreterMap = make(map[string]*InterpreterConfig)
}

func expandConfigPaths() {
	home := os.Getenv("HOME")
	expandList := func(paths []string) []string {
		result := make([]string, len(paths))
		for i, p := range paths {
			result[i] = strings.ReplaceAll(p, "$HOME", home)
		}
		return result
	}

	config.Paths.Sensitive = expandList(config.Paths.Sensitive)
	config.Paths.AlwaysAllowed = expandList(config.Paths.AlwaysAllowed)
	config.Paths.SandboxBlocked = expandList(config.Paths.SandboxBlocked)
	config.Prompt.LogFile = strings.ReplaceAll(config.Prompt.LogFile, "$HOME", home)
}

// ============================================================================
// Bash command handling
// ============================================================================

func handleBash(args map[string]interface{}) {
	cmd, _ := args["cmd"].(string)
	if cmd == "" {
		logDecision("ALLOW", "Bash", "empty command")
		os.Exit(ExitAllow)
	}

	currentCmd = cmd // Store for prompt context

	// 1. Check raw reject patterns BEFORE parsing (fork bombs, injection attacks)
	for _, pattern := range config.Patterns.RejectRaw {
		if strings.Contains(cmd, pattern) {
			logDecision("BLOCK", "Bash", fmt.Sprintf("reject_raw pattern: %s", pattern))
			sendAutoBlockNotification(pattern)
			os.Exit(ExitReject)
		}
	}

	// 2. Parse command
	result := parseCommand(cmd)
	if result.Error != "" {
		// Parse error - prompt user to verify manually
		if promptUser("PARSE ERROR", fmt.Sprintf("Could not parse command:\n%s\n\nError: %s\n\nAllow anyway?", cmd, result.Error)) {
			logDecision("ALLOW", "Bash", fmt.Sprintf("user approved despite parse error: %s", result.Error))
			os.Exit(ExitAllow)
		} else {
			logDecision("DENY", "Bash", fmt.Sprintf("user denied after parse error: %s", result.Error))
			os.Exit(ExitReject)
		}
	}

	// 3. Check reject patterns against parsed commands (command name only, not string literals)
	for _, pattern := range config.Patterns.Reject {
		for _, cmdInfo := range result.Commands {
			// Match against command name, not full command with string args
			if strings.Contains(cmdInfo.Name, pattern) {
				logDecision("BLOCK", "Bash", fmt.Sprintf("reject pattern: %s in command: %s", pattern, cmdInfo.Name))
				sendAutoBlockNotification(pattern)
				os.Exit(ExitReject)
			}
			// Also check if pattern matches "cmd arg" style (e.g., "xargs rm")
			// But only check the command name + first few args, not quoted strings
			cmdWithArgs := cmdInfo.Name
			for _, arg := range cmdInfo.Args[1:] { // skip first arg (command name)
				// Skip if arg looks like a quoted string (contains spaces or quotes)
				if !strings.ContainsAny(arg, " \"'") {
					cmdWithArgs += " " + arg
				}
			}
			if strings.Contains(cmdWithArgs, pattern) {
				logDecision("BLOCK", "Bash", fmt.Sprintf("reject pattern: %s in command: %s", pattern, cmdWithArgs))
				sendAutoBlockNotification(pattern)
				os.Exit(ExitReject)
			}
		}
	}

	// 4. Check risky patterns from AST analysis
	if len(result.RiskyPatterns) > 0 {
		detail := result.RiskyPatterns[0].Detail
		logDecision("BLOCK", "Bash", fmt.Sprintf("risky: %s", detail))
		sendAutoBlockNotification(detail)
		os.Exit(ExitReject)
	}

	// 5. Check pipes to interpreters
	for _, pipe := range result.PipesToInterp {
		if pipe.RequiresPrompt {
			msg := fmt.Sprintf("Piping to %s\nSource: %s", pipe.Interpreter, pipe.SourceType)
			if len(pipe.MatchedPatterns) > 0 {
				msg += fmt.Sprintf("\nMatched patterns: %s", strings.Join(pipe.MatchedPatterns, ", "))
			}
			msg += fmt.Sprintf("\n\n%s", cmd)

			if promptUser(fmt.Sprintf("PIPE TO %s", pipe.Interpreter), msg) {
				logDecision("ALLOW", "Bash", fmt.Sprintf("user approved pipe to %s", pipe.Interpreter))
				os.Exit(ExitAllow)
			} else {
				logDecision("DENY", "Bash", fmt.Sprintf("user denied pipe to %s", pipe.Interpreter))
				os.Exit(ExitReject)
			}
		} else {
			logDecision("ALLOW", "Bash", fmt.Sprintf("pipe to %s - no risky patterns", pipe.Interpreter))
			os.Exit(ExitAllow)
		}
	}

	// 4. Check heredocs
	for _, heredoc := range result.Heredocs {
		if heredoc.RequiresPrompt {
			msg := fmt.Sprintf("Heredoc to %s", heredoc.Interpreter)
			if len(heredoc.MatchedPatterns) > 0 {
				msg += fmt.Sprintf("\nMatched patterns: %s", strings.Join(heredoc.MatchedPatterns, ", "))
			}
			msg += fmt.Sprintf("\n\n%s", cmd)

			if promptUser("HEREDOC", msg) {
				logDecision("ALLOW", "Bash", fmt.Sprintf("user approved heredoc to %s", heredoc.Interpreter))
				os.Exit(ExitAllow)
			} else {
				logDecision("DENY", "Bash", fmt.Sprintf("user denied heredoc to %s", heredoc.Interpreter))
				os.Exit(ExitReject)
			}
		}
	}

	// 5. Check always-ask patterns (regex) against each parsed command AND raw cmd
	// (raw cmd catches redirects like "echo > .env" which aren't in parsed FullCmd)
	for _, re := range alwaysAskRegexps {
		matched := re.MatchString(cmd) // Check raw command (catches redirects)
		if !matched {
			for _, cmdInfo := range result.Commands {
				if re.MatchString(cmdInfo.FullCmd) {
					matched = true
					break
				}
			}
		}
		if matched {
			if promptUser("CONFIRM", fmt.Sprintf("Pattern: %s\n\nCmd: %s", re.String(), cmd)) {
				logDecision("ALLOW", "Bash", fmt.Sprintf("user approved pattern: %s", re.String()))
				os.Exit(ExitAllow)
			} else {
				logDecision("DENY", "Bash", fmt.Sprintf("user denied pattern: %s", re.String()))
				os.Exit(ExitReject)
			}
		}
	}

	// 6. Check sensitive paths (before read-only check)
	allPaths := gatherAllPaths(result)
	for _, p := range allPaths {
		resolved := resolvePath(p)
		if sensitive := matchesSensitivePath(resolved); sensitive != "" {
			if promptUser("SENSITIVE", fmt.Sprintf("Path: %s\nLocation: %s\n\nCmd: %s", p, sensitive, cmd)) {
				logDecision("ALLOW", "Bash", fmt.Sprintf("user approved sensitive: %s", p))
				os.Exit(ExitAllow)
			} else {
				logDecision("DENY", "Bash", fmt.Sprintf("user denied sensitive: %s", p))
				os.Exit(ExitReject)
			}
		}
	}

	// 7. Read-only commands: allow
	if isReadOnly(result) {
		logDecision("ALLOW", "Bash", "read-only command")
		os.Exit(ExitAllow)
	}

	// 8. Check script paths for risky content
	for _, sp := range result.ScriptPaths {
		resolved := resolvePath(sp)
		if risk := scanScriptFile(resolved, 0, nil); risk != "" {
			if promptUser("SCRIPT RISK", fmt.Sprintf("%s\n\nCmd: %s", risk, cmd)) {
				logDecision("ALLOW", "Bash", "user approved risky script")
				os.Exit(ExitAllow)
			} else {
				logDecision("DENY", "Bash", "user denied risky script")
				os.Exit(ExitReject)
			}
		}
	}

	// 9. Skip PWD check for non-destructive commands (cp, ln, touch, etc.)
	if isNonDestructive(result) {
		logDecision("ALLOW", "Bash", "non-destructive command")
		os.Exit(ExitAllow)
	}

	// 10. Check paths against PWD
	var outsidePaths []string
	for _, p := range allPaths {
		resolved := resolvePath(p)
		if !isUnderPWD(resolved) && !isAlwaysAllowed(resolved) {
			outsidePaths = append(outsidePaths, p)
		}
	}

	if len(outsidePaths) == 0 {
		logDecision("ALLOW", "Bash", "all paths in PWD or allowed")
		os.Exit(ExitAllow)
	}

	// 10. Check if outside paths are just files (not dirs)
	var riskyPaths []string
	for _, p := range outsidePaths {
		resolved := resolvePath(p)
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			continue // File or doesn't exist - generally safe
		}
		riskyPaths = append(riskyPaths, p)
	}

	if len(riskyPaths) == 0 {
		logDecision("ALLOW", "Bash", "single files only")
		os.Exit(ExitAllow)
	}

	// 11. Directory operations outside PWD - prompt
	if promptUser("DIRECTORY OP", fmt.Sprintf("Paths: %s\nPWD: %s\n\nCmd: %s",
		strings.Join(riskyPaths, ", "), workDir, cmd)) {
		logDecision("ALLOW", "Bash", "user approved directory op")
		os.Exit(ExitAllow)
	} else {
		logDecision("DENY", "Bash", "user denied directory op")
		os.Exit(ExitReject)
	}
}

func handleFileTool(args map[string]interface{}) {
	path, _ := args["path"].(string)
	if path == "" {
		logDecision("ALLOW", toolName, "no path")
		os.Exit(ExitAllow)
	}

	resolved := resolvePath(path)
	if sensitive := matchesSensitivePath(resolved); sensitive != "" {
		if promptUser("SENSITIVE FILE", fmt.Sprintf("Path: %s\nLocation: %s", path, sensitive)) {
			logDecision("ALLOW", toolName, fmt.Sprintf("user approved: %s", path))
			os.Exit(ExitAllow)
		} else {
			logDecision("DENY", toolName, fmt.Sprintf("user denied: %s", path))
			os.Exit(ExitReject)
		}
	}

	logDecision("ALLOW", toolName, path)
	os.Exit(ExitAllow)
}

// ============================================================================
// Path helpers
// ============================================================================

func resolvePath(path string) string {
	// Expand ~ and $HOME
	home := os.Getenv("HOME")
	if strings.HasPrefix(path, "~") {
		path = home + path[1:]
	}
	path = strings.ReplaceAll(path, "$HOME", home)

	// Expand other env vars
	path = expandEnvVars(path)

	// Resolve to absolute path
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	// Resolve symlinks
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}

	// If file doesn't exist, resolve parent
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Join(resolved, base)
	}

	return path
}

func expandEnvVars(s string) string {
	// Handle common env vars directly
	commonVars := []string{"HOME", "USER", "TMPDIR", "PWD", "PATH"}
	for _, v := range commonVars {
		s = strings.ReplaceAll(s, "$"+v, os.Getenv(v))
		s = strings.ReplaceAll(s, "${"+v+"}", os.Getenv(v))
	}

	// For complex expansions, fall back to bash
	if strings.Contains(s, "$") {
		if expanded := bashExpand(s); expanded != "" {
			return expanded
		}
	}

	return s
}

func bashExpand(s string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("set -f; echo %q", s))
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func isUnderPWD(path string) bool {
	pwdResolved := resolvePath(workDir)
	return strings.HasPrefix(path, pwdResolved)
}

func isAlwaysAllowed(path string) bool {
	for _, allowed := range config.Paths.AlwaysAllowed {
		if strings.HasPrefix(path, allowed) {
			return true
		}
	}
	return false
}

func matchesSensitivePath(path string) string {
	for _, sensitive := range config.Paths.Sensitive {
		if strings.HasPrefix(path, sensitive) {
			return sensitive
		}
	}
	return ""
}

func gatherAllPaths(result *ParseResult) []string {
	paths := make(map[string]bool)
	for _, p := range result.LiteralPaths {
		paths[p] = true
	}
	for _, v := range result.VarReferences {
		if expanded := expandEnvVars(v); expanded != v {
			paths[expanded] = true
		}
	}
	var out []string
	for p := range paths {
		out = append(out, p)
	}
	return out
}

func isReadOnly(result *ParseResult) bool {
	for _, cmd := range result.Commands {
		if writeCommands[cmd.Name] {
			return false
		}
	}
	return true
}

func isNonDestructive(result *ParseResult) bool {
	for _, cmd := range result.Commands {
		if writeCommands[cmd.Name] && !nonDestructiveCmds[cmd.Name] {
			return false
		}
	}
	return true
}

// ============================================================================
// Script scanning
// ============================================================================

func scanScriptFile(path string, depth int, visited map[string]bool) string {
	if depth >= config.ScriptScan.MaxDepth {
		return ""
	}

	if visited == nil {
		visited = make(map[string]bool)
	}
	if visited[path] {
		return ""
	}
	visited[path] = true

	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	if info.Size() > int64(config.ScriptScan.MaxBytes) {
		return fmt.Sprintf("script too large (%d bytes)", info.Size())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	content := string(data)

	// Check reject patterns
	for _, pattern := range config.Patterns.Reject {
		if strings.Contains(content, pattern) {
			return fmt.Sprintf("script:%s contains '%s'", path, pattern)
		}
	}

	// Check always-ask patterns
	for _, re := range alwaysAskRegexps {
		if re.MatchString(content) {
			return fmt.Sprintf("script:%s matches '%s'", path, re.String())
		}
	}

	return ""
}

// ============================================================================
// User prompting
// ============================================================================

// promptUser delegates to external handler if configured, otherwise uses built-in methods
// External handler receives JSON on stdin: {"title": "...", "message": "...", "context": {...}}
// Handler must exit 0 for allow, non-zero for deny
func promptUser(title, message string) bool {
	fullTitle := fmt.Sprintf("[%s] %s", tmuxContext, title)

	// Check for external handler override
	handler := os.Getenv("AMP_PERMISSION_PROMPT_HANDLER")
	if handler == "" {
		handler = filepath.Join(os.Getenv("HOME"), ".local/bin/amp-prompt-handler")
	}

	// Try external handler if it exists and is executable
	if info, err := os.Stat(handler); err == nil && (info.Mode()&0111) != 0 {
		if result := promptExternal(handler, fullTitle, message); result != nil {
			return *result
		}
	}

	// Fallback to built-in osascript (macOS)
	if result := promptOsascript(fullTitle, message); result != nil {
		return *result
	}

	// TTY fallback
	if result := promptTTY(fullTitle, message); result != nil {
		return *result
	}

	logDecision("NO_PROMPT_METHOD", toolName, "")
	return false
}

// promptExternal calls an external handler binary with JSON payload
// Handler exit codes: 0 = allow, 2 = reject
// Returns nil if handler failed to run (fallback to built-in)
func promptExternal(handler, title, message string) *bool {
	payload := map[string]interface{}{
		"title":   title,
		"message": message,
		"context": map[string]string{
			"cmd":  currentCmd,
			"pwd":  workDir,
			"tool": toolName,
			"tmux": tmuxContext,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil
	}

	cmd := exec.Command(handler)
	cmd.Stdin = strings.NewReader(string(jsonData))
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	allowed := err == nil // exit 0 = allow, anything else = reject
	return &allowed
}

func promptOsascript(title, message string) *bool {
	// Escape for AppleScript
	message = strings.ReplaceAll(message, "\\", "\\\\")
	message = strings.ReplaceAll(message, "\"", "\\\"")
	message = strings.ReplaceAll(message, "\n", "\\n")
	title = strings.ReplaceAll(title, "\"", "\\\"")

	buttons := `{"Deny", "Allow"}`
	if os.Getenv("TMUX") != "" {
		buttons = `{"Deny", "Visit", "Allow"}`
	}

	timeout := config.Prompt.TimeoutSecs
	if timeout == 0 {
		timeout = 60
	}

	for {
		script := fmt.Sprintf(`
			set dialogResult to display dialog "%s" with title "%s" buttons %s default button "Deny" with icon caution giving up after %d
			return (button returned of dialogResult) & "|" & (gave up of dialogResult)
		`, message, title, buttons, timeout)

		cmd := exec.Command("osascript", "-e", script)
		out, _ := cmd.Output()
		result := strings.TrimSpace(string(out))

		parts := strings.Split(result, "|")
		button := ""
		gaveUp := false
		if len(parts) >= 1 {
			button = parts[0]
		}
		if len(parts) >= 2 {
			gaveUp = parts[1] == "true"
		}

		if button == "Visit" {
			visitTmuxPane()
			time.Sleep(300 * time.Millisecond)
			continue
		}

		if gaveUp {
			sendTimeoutNotification(title)
			logDecision("TIMEOUT", toolName, "")
		}

		allowed := button == "Allow"
		return &allowed
	}
}

func promptTTY(title, message string) *bool {
	tty := "/dev/tty"
	if pane := os.Getenv("TMUX_PANE"); pane != "" {
		if out, err := exec.Command("tmux", "display-message", "-t", pane, "-p", "#{pane_tty}").Output(); err == nil {
			if t := strings.TrimSpace(string(out)); t != "" {
				tty = t
			}
		}
	}

	f, err := os.OpenFile(tty, os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	defer f.Close()

	fmt.Fprintf(f, "\n=== AMP PERMISSION ===\n")
	fmt.Fprintf(f, "%s\n\n", title)
	fmt.Fprintf(f, "%s\n\n", message)
	fmt.Fprintf(f, "Allow? [y/N]: ")

	buf := make([]byte, 10)
	n, _ := f.Read(buf)
	response := strings.TrimSpace(string(buf[:n]))

	allowed := strings.ToLower(response) == "y"
	return &allowed
}

func visitTmuxPane() {
	if os.Getenv("TMUX") == "" {
		return
	}

	pane := os.Getenv("TMUX_PANE")
	var target string
	if pane != "" {
		out, _ := exec.Command("tmux", "display-message", "-t", pane, "-p", "#S:#I").Output()
		target = strings.TrimSpace(string(out))
	} else {
		out, _ := exec.Command("tmux", "display-message", "-p", "#S:#I").Output()
		target = strings.TrimSpace(string(out))
	}

	exec.Command("tmux", "switch-client", "-t", target).Run()
	exec.Command("tmux", "select-window", "-t", target).Run()

	// Bring terminal to front
	for _, app := range []string{"WezTerm", "iTerm", "Alacritty", "kitty", "Terminal"} {
		if exec.Command("pgrep", "-qf", app).Run() == nil {
			exec.Command("open", "-a", app).Run()
			break
		}
	}
}

func sendTimeoutNotification(title string) {
	notifTitle := fmt.Sprintf("Amp Permission [%s]", tmuxContext)
	exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "Timed out - auto denied" with title "%s" sound name "Basso"`, notifTitle)).Run()
}

func sendAutoBlockNotification(pattern string) {
	notifTitle := fmt.Sprintf("Amp Permission [%s]", tmuxContext)
	exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "Auto-blocked: %s" with title "%s" sound name "Basso"`, pattern, notifTitle)).Run()
}

// ============================================================================
// Logging
// ============================================================================

func getTmuxContext() string {
	if os.Getenv("TMUX") == "" {
		return "no-tmux"
	}

	pane := os.Getenv("TMUX_PANE")
	var out []byte
	if pane != "" {
		out, _ = exec.Command("tmux", "display-message", "-t", pane, "-p", "#S:#W(#I)").Output()
	} else {
		out, _ = exec.Command("tmux", "display-message", "-p", "#S:#W(#I)").Output()
	}
	if ctx := strings.TrimSpace(string(out)); ctx != "" {
		return ctx
	}
	return "unknown"
}

func logDecision(decision, tool, detail string) {
	logFile := config.Prompt.LogFile
	if logFile == "" {
		logFile = filepath.Join(os.Getenv("HOME"), ".config/amp-permissions/decisions.log")
	}

	os.MkdirAll(filepath.Dir(logFile), 0755)

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] [%s] %s | %s | %s\n", timestamp, tmuxContext, decision, tool, detail)

	// Truncate if too long
	maxLines := config.Prompt.MaxLogLines
	if maxLines == 0 {
		maxLines = 100
	}
	truncateLog(logFile, maxLines)
}

func truncateLog(path string, maxLines int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) <= maxLines {
		return
	}

	lines = lines[len(lines)-maxLines:]
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// ============================================================================
// AST Parsing (from amp-permission-parser)
// ============================================================================

type ParseResult struct {
	Commands      []CommandInfo  `json:"commands"`
	LiteralPaths  []string       `json:"literal_paths"`
	VarReferences []string       `json:"var_references"`
	ScriptPaths   []string       `json:"script_paths"`
	Heredocs      []HeredocInfo  `json:"heredocs,omitempty"`
	PipesToInterp []PipeInfo     `json:"pipes_to_interp,omitempty"`
	RiskyPatterns []RiskyPattern `json:"risky_patterns,omitempty"`
	Error         string         `json:"error,omitempty"`
}

type CommandInfo struct {
	Name     string   `json:"name"`
	Args     []string `json:"args"`
	FullCmd  string   `json:"full_cmd"`
	IsRisky  bool     `json:"is_risky,omitempty"`
	RiskType string   `json:"risk_type,omitempty"`
}

type HeredocInfo struct {
	Delimiter       string   `json:"delimiter"`
	Content         string   `json:"content,omitempty"`
	Interpreter     string   `json:"interpreter,omitempty"`
	MatchedPatterns []string `json:"matched_patterns,omitempty"`
	RequiresPrompt  bool     `json:"requires_prompt"`
}

type PipeInfo struct {
	Interpreter     string   `json:"interpreter"`
	FullPipe        string   `json:"full_pipe"`
	SourceType      string   `json:"source_type"`
	SourcePath      string   `json:"source_path,omitempty"`
	MatchedPatterns []string `json:"matched_patterns,omitempty"`
	RequiresPrompt  bool     `json:"requires_prompt"`
}

type RiskyPattern struct {
	Type    string `json:"type"`
	Detail  string `json:"detail"`
	Command string `json:"command"`
}

var pathRegex = regexp.MustCompile(`^[~/]|^\$HOME`)

func parseCommand(cmdStr string) *ParseResult {
	result := &ParseResult{
		Commands:      []CommandInfo{},
		LiteralPaths:  []string{},
		VarReferences: []string{},
		ScriptPaths:   []string{},
	}

	reader := strings.NewReader(cmdStr)
	parser := syntax.NewParser(syntax.Variant(syntax.LangBash))

	file, err := parser.Parse(reader, "")
	if err != nil {
		result.Error = fmt.Sprintf("parse error: %v", err)
		return result
	}

	seenPaths := make(map[string]bool)
	seenVars := make(map[string]bool)
	seenScripts := make(map[string]bool)

	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CallExpr:
			info := extractCallExpr(n)
			if info != nil {
				result.Commands = append(result.Commands, *info)
				checkRiskyPatterns(result, info)
				extractScriptPath(result, info, seenScripts)
			}

		case *syntax.Redirect:
			if n.Hdoc != nil {
				heredoc := extractHeredocWithScan(n, file)
				if heredoc != nil {
					result.Heredocs = append(result.Heredocs, *heredoc)
				}
			}

		case *syntax.BinaryCmd:
			if n.Op == syntax.Pipe {
				checkPipeToInterpreterWithScan(result, n)
			}

		case *syntax.Word:
			for _, part := range n.Parts {
				extractFromWordPart(part, seenPaths, seenVars, result)
			}
		}
		return true
	})

	return result
}

func extractCallExpr(call *syntax.CallExpr) *CommandInfo {
	if len(call.Args) == 0 {
		return nil
	}

	printer := syntax.NewPrinter()
	var fullCmd strings.Builder
	printer.Print(&fullCmd, call)

	var args []string
	for _, arg := range call.Args {
		var sb strings.Builder
		printer.Print(&sb, arg)
		args = append(args, sb.String())
	}

	name := filepath.Base(args[0])

	return &CommandInfo{
		Name:    name,
		Args:    args,
		FullCmd: fullCmd.String(),
	}
}

func extractFromWordPart(part syntax.WordPart, seenPaths, seenVars map[string]bool, result *ParseResult) {
	switch p := part.(type) {
	case *syntax.Lit:
		val := p.Value
		if pathRegex.MatchString(val) && !seenPaths[val] {
			seenPaths[val] = true
			result.LiteralPaths = append(result.LiteralPaths, val)
		}

	case *syntax.ParamExp:
		varName := "$" + p.Param.Value
		if !p.Short {
			varName = "${" + p.Param.Value + "}"
		}
		if !seenVars[varName] {
			seenVars[varName] = true
			result.VarReferences = append(result.VarReferences, varName)
		}

	case *syntax.DblQuoted:
		for _, subPart := range p.Parts {
			extractFromWordPart(subPart, seenPaths, seenVars, result)
		}

	case *syntax.CmdSubst:
		for _, stmt := range p.Stmts {
			if stmt.Cmd != nil {
				if call, ok := stmt.Cmd.(*syntax.CallExpr); ok {
					info := extractCallExpr(call)
					if info != nil {
						result.Commands = append(result.Commands, *info)
					}
				}
			}
		}
	}
}

func extractHeredocWithScan(redir *syntax.Redirect, file *syntax.File) *HeredocInfo {
	if redir.Hdoc == nil {
		return nil
	}

	printer := syntax.NewPrinter()
	var delim strings.Builder
	printer.Print(&delim, redir.Word)

	var content strings.Builder
	printer.Print(&content, redir.Hdoc)

	heredocContent := content.String()

	var interpreter string
	syntax.Walk(file, func(node syntax.Node) bool {
		if stmt, ok := node.(*syntax.Stmt); ok {
			for _, r := range stmt.Redirs {
				if r == redir {
					if call, ok := stmt.Cmd.(*syntax.CallExpr); ok && len(call.Args) > 0 {
						var cmdName strings.Builder
						printer.Print(&cmdName, call.Args[0])
						interpreter = filepath.Base(cmdName.String())
					}
					return false
				}
			}
		}
		return true
	})

	info := &HeredocInfo{
		Delimiter:      delim.String(),
		Content:        heredocContent,
		Interpreter:    interpreter,
		RequiresPrompt: false,
	}

	if interpreter != "" {
		if interpInfo := interpreterMap[interpreter]; interpInfo != nil {
			matchedPatterns := scanContentForPatterns(heredocContent, interpInfo.Patterns)
			if len(matchedPatterns) > 0 {
				info.MatchedPatterns = matchedPatterns
				info.RequiresPrompt = true
			}
		}
	}

	return info
}

func checkPipeToInterpreterWithScan(result *ParseResult, binary *syntax.BinaryCmd) {
	if binary.Op != syntax.Pipe {
		return
	}

	printer := syntax.NewPrinter()

	if binary.Y == nil || binary.Y.Cmd == nil {
		return
	}

	if call, ok := binary.Y.Cmd.(*syntax.CallExpr); ok {
		if len(call.Args) > 0 {
			var cmdName strings.Builder
			printer.Print(&cmdName, call.Args[0])
			name := filepath.Base(cmdName.String())

			if interpInfo := interpreterMap[name]; interpInfo != nil {
				var fullPipe strings.Builder
				printer.Print(&fullPipe, binary)

				// Extract interpreter args (everything after the command name)
				var interpArgs strings.Builder
				for i, arg := range call.Args[1:] {
					if i > 0 {
						interpArgs.WriteString(" ")
					}
					printer.Print(&interpArgs, arg)
				}
				interpArgsStr := interpArgs.String()

				pipeInfo := PipeInfo{
					Interpreter:    name,
					FullPipe:       fullPipe.String(),
					RequiresPrompt: false,
				}

				sourceType, sourcePath := analyzePipeSource(binary.X, printer)
				pipeInfo.SourceType = sourceType
				pipeInfo.SourcePath = sourcePath

				// First check interpreter args for dangerous patterns (e.g., awk 'system("rm")')
				if len(interpInfo.Patterns) > 0 && interpArgsStr != "" {
					matchedPatterns := scanContentForPatterns(interpArgsStr, interpInfo.Patterns)
					if len(matchedPatterns) > 0 {
						pipeInfo.MatchedPatterns = matchedPatterns
						pipeInfo.RequiresPrompt = true
					}
				}

				// If no pattern matched in args, check based on source
				if !pipeInfo.RequiresPrompt {
					switch sourceType {
					case "network":
						// Network source always prompts (can't scan remote content)
						pipeInfo.RequiresPrompt = true

					case "local_file":
						// Scan local file content for patterns
						content, err := readLocalFile(sourcePath)
						if err == nil {
							matchedPatterns := scanContentForPatterns(content, interpInfo.Patterns)
							if len(matchedPatterns) > 0 {
								pipeInfo.MatchedPatterns = matchedPatterns
								pipeInfo.RequiresPrompt = true
							}
						}
						// If file can't be read, don't prompt - it's probably fine

					case "command":
						// Command output piped to interpreter - only prompt if patterns found in args
						// (already checked above)

					default:
						// Unknown source - only prompt if patterns found
					}
				}

				result.PipesToInterp = append(result.PipesToInterp, pipeInfo)
			}
		}
	}
}

func analyzePipeSource(stmt *syntax.Stmt, printer *syntax.Printer) (sourceType, sourcePath string) {
	if stmt == nil || stmt.Cmd == nil {
		return "unknown", ""
	}

	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Args) == 0 {
		return "command", ""
	}

	var cmdName strings.Builder
	printer.Print(&cmdName, call.Args[0])
	name := filepath.Base(cmdName.String())

	if networkCommands[name] {
		return "network", ""
	}

	if name == "cat" || name == "head" || name == "tail" || name == "less" || name == "more" {
		if len(call.Args) > 1 {
			var pathArg strings.Builder
			printer.Print(&pathArg, call.Args[len(call.Args)-1])
			path := pathArg.String()
			if !strings.Contains(path, "://") && !strings.HasPrefix(path, "-") {
				return "local_file", resolvePath(path)
			}
		}
	}

	return "command", ""
}

func readLocalFile(path string) (string, error) {
	maxSize := config.ScriptScan.MaxBytes
	if maxSize == 0 {
		maxSize = 102400
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > int64(maxSize) {
		return "", fmt.Errorf("file too large")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func scanContentForPatterns(content string, patterns []string) []string {
	var matched []string
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			if re.MatchString(content) {
				matched = append(matched, pattern)
			}
		} else {
			if strings.Contains(content, pattern) {
				matched = append(matched, pattern)
			}
		}
	}
	return matched
}

func checkRiskyPatterns(result *ParseResult, info *CommandInfo) {
	name := info.Name

	// 1. Check dangerous targets (from config) - any arg matching blocks
	if writeCommands[name] {
		for _, arg := range info.Args[1:] { // skip command name
			if dangerousTargets[arg] {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "dangerous_target",
					Detail:  fmt.Sprintf("%s targeting %s", name, arg),
					Command: info.FullCmd,
				})
				return
			}
		}
		// Also check literal_paths extracted from subqueries like $(echo ~)
		for _, p := range result.LiteralPaths {
			if dangerousTargets[p] {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "dangerous_target_subquery",
					Detail:  fmt.Sprintf("%s with dangerous path %s in subquery", name, p),
					Command: info.FullCmd,
				})
				return
			}
		}
	}

	// 2. Check dangerous flags (from config)
	if flags, ok := dangerousFlags[name]; ok {
		argsJoined := strings.Join(info.Args, " ")
		for _, flag := range flags {
			if strings.Contains(argsJoined, flag) {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "dangerous_flag",
					Detail:  fmt.Sprintf("%s with %s", name, flag),
					Command: info.FullCmd,
				})
				return
			}
		}
	}

	// 3. Special case: find -exec with write commands
	if name == "find" {
		checkFindPatterns(result, info)
	}
}

func checkFindPatterns(result *ParseResult, info *CommandInfo) {
	args := info.Args

	for i, arg := range args {
		// Check if this is a find_exec_flag from config (e.g., -exec, -execdir, -ok, -okdir)
		if findExecFlags[arg] {
			if i+1 < len(args) {
				execCmd := filepath.Base(args[i+1])
				if writeCommands[execCmd] {
					result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
						Type:    "find_exec_write",
						Detail:  fmt.Sprintf("find %s %s (write command)", arg, execCmd),
						Command: info.FullCmd,
					})
					return
				}
			}
		}
	}
}

func extractScriptPath(result *ParseResult, info *CommandInfo, seen map[string]bool) {
	name := info.Name

	if interpreterMap[name] != nil || name == "source" || name == "." {
		for _, arg := range info.Args[1:] {
			if !strings.HasPrefix(arg, "-") {
				if !seen[arg] {
					seen[arg] = true
					result.ScriptPaths = append(result.ScriptPaths, arg)
				}
				break
			}
		}
	}

	for _, arg := range info.Args {
		if strings.HasPrefix(arg, "./") && !strings.HasPrefix(arg, "-") {
			if !seen[arg] {
				seen[arg] = true
				result.ScriptPaths = append(result.ScriptPaths, arg)
			}
		}
	}
}

// ============================================================================
// CLI commands
// ============================================================================

func runTest() {
	jsonArg := "{}"
	tool := "Bash"
	if len(os.Args) > 2 {
		jsonArg = os.Args[2]
	}
	if len(os.Args) > 3 {
		tool = os.Args[3]
	}

	toolName = tool
	workDir, _ = os.Getwd()
	tmuxContext = getTmuxContext()

	fmt.Printf("Testing: %s\n", tool)
	fmt.Printf("Args: %s\n", jsonArg)
	fmt.Printf("PWD: %s\n", workDir)
	fmt.Println("---")

	var args map[string]interface{}
	json.Unmarshal([]byte(jsonArg), &args)

	switch tool {
	case "Bash":
		handleBash(args)
	case "edit_file", "create_file":
		handleFileTool(args)
	default:
		fmt.Println("Unknown tool")
		os.Exit(1)
	}
}

func runParse() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: amp-permission --parse 'command'")
		os.Exit(1)
	}

	result := parseCommand(os.Args[2])
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func printHelp() {
	fmt.Println(`amp-permission - Unified Amp permission evaluator

DELEGATE MODE (called by Amp):
  Receives AGENT_TOOL_NAME env var and JSON args on stdin

CLI MODE:
  --test JSON [TOOL]  Test a command without executing
  --parse CMD         Show parsed AST for a command  
  --help              Show this help

CONFIG:
  ~/.config/amp-permissions/permissions.yaml

EXIT CODES:
  0 = allow
  1 = ask (user denied or timeout)
  2 = reject (blocked by policy)`)
}
