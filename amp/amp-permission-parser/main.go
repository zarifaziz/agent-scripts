// amp-permission-parser - Parse shell commands using mvdan.cc/sh AST
//
// Outputs JSON analysis of commands including:
// - All commands (including in pipes, subshells, command substitutions)
// - Literal paths, variable references, script paths
// - Heredocs, pipes to interpreters
// - Risky patterns like rm -rf
//
// Build: go build -o amp-permission-parser
// Usage: echo 'rm -rf /tmp/foo' | ./amp-permission-parser

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type CommandInfo struct {
	Name      string   `json:"name"`
	Args      []string `json:"args"`
	FullCmd   string   `json:"full_cmd"`
	IsRisky   bool     `json:"is_risky,omitempty"`
	RiskType  string   `json:"risk_type,omitempty"`
}

type ParseResult struct {
	Commands          []CommandInfo    `json:"commands"`
	LiteralPaths      []string         `json:"literal_paths"`
	VarReferences     []string         `json:"var_references"`
	ScriptPaths       []string         `json:"script_paths"`
	Heredocs          []HeredocInfo    `json:"heredocs,omitempty"`
	PipesToInterp     []PipeInfo       `json:"pipes_to_interp,omitempty"`
	RiskyPatterns     []RiskyPattern   `json:"risky_patterns,omitempty"`
	AlwaysAsk         []AlwaysAskMatch `json:"always_ask,omitempty"`
	IsReadOnly        bool             `json:"is_read_only"`
	Error             string           `json:"error,omitempty"`
}

type AlwaysAskMatch struct {
	Pattern string `json:"pattern"`
	Command string `json:"command"`
}

type HeredocInfo struct {
	Delimiter string `json:"delimiter"`
	Content   string `json:"content,omitempty"`
}

type PipeInfo struct {
	Interpreter string `json:"interpreter"`
	FullPipe    string `json:"full_pipe"`
}

type RiskyPattern struct {
	Type    string `json:"type"`
	Detail  string `json:"detail"`
	Command string `json:"command"`
}

var defaultInterpreters = []string{
	"bash", "sh", "zsh", "python", "python3", "node", "ruby", "perl", "bun",
}

var interpreters map[string]bool

func init() {
	interpreters = make(map[string]bool)
	
	// Try loading from config file
	configPath := filepath.Join(os.Getenv("HOME"), ".config/amp-permissions/interpreters.txt")
	if data, err := os.ReadFile(configPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				interpreters[line] = true
			}
		}
	}
	
	// Fallback to defaults if config empty/missing
	if len(interpreters) == 0 {
		for _, i := range defaultInterpreters {
			interpreters[i] = true
		}
	}
}

var pathRegex = regexp.MustCompile(`^[~/]|^\$HOME`)
var homeExpansion = regexp.MustCompile(`^\$HOME`)

var alwaysAskPatterns []*regexp.Regexp
var rejectPatterns []string
var writeCommands map[string]bool

func loadConfigPatterns() {
	configDir := filepath.Join(os.Getenv("HOME"), ".config/amp-permissions")
	
	// Load always-ask patterns (regex)
	if data, err := os.ReadFile(filepath.Join(configDir, "always-ask-patterns.txt")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				if re, err := regexp.Compile(line); err == nil {
					alwaysAskPatterns = append(alwaysAskPatterns, re)
				}
			}
		}
	}
	
	// Load reject patterns (substring match)
	if data, err := os.ReadFile(filepath.Join(configDir, "reject-patterns.txt")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				rejectPatterns = append(rejectPatterns, line)
			}
		}
	}
	
	// Load write commands (non-readonly)
	writeCommands = make(map[string]bool)
	if data, err := os.ReadFile(filepath.Join(configDir, "write-commands.txt")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				writeCommands[line] = true
			}
		}
	}
}

func main() {
	loadConfigPatterns()
	
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		outputError(fmt.Sprintf("failed to read stdin: %v", err))
		return
	}

	cmdStr := strings.TrimSpace(string(input))
	if cmdStr == "" {
		outputResult(ParseResult{})
		return
	}

	result := parseCommand(cmdStr)
	
	// Determine if command is read-only (no write commands found)
	result.IsReadOnly = true
	for _, cmd := range result.Commands {
		if writeCommands[cmd.Name] {
			result.IsReadOnly = false
			break
		}
	}
	
	// Check reject patterns from config (substring match)
	for _, pattern := range rejectPatterns {
		if strings.Contains(cmdStr, pattern) {
			result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
				Type:    "reject_pattern",
				Detail:  fmt.Sprintf("matches reject pattern: %s", pattern),
				Command: cmdStr,
			})
			break // One match is enough to reject
		}
	}
	
	// Check always-ask patterns against full command
	for _, re := range alwaysAskPatterns {
		if re.MatchString(cmdStr) {
			result.AlwaysAsk = append(result.AlwaysAsk, AlwaysAskMatch{
				Pattern: re.String(),
				Command: cmdStr,
			})
		}
	}
	
	outputResult(result)
}

func parseCommand(cmdStr string) ParseResult {
	result := ParseResult{
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
				checkRiskyPatterns(&result, info)
				extractScriptPath(&result, info, seenScripts)
			}

		case *syntax.Redirect:
			if n.Hdoc != nil {
				heredoc := extractHeredoc(n)
				if heredoc != nil {
					result.Heredocs = append(result.Heredocs, *heredoc)
				}
			}

		case *syntax.BinaryCmd:
			if n.Op == syntax.Pipe {
				checkPipeToInterpreter(&result, n)
			}

		case *syntax.Word:
			for _, part := range n.Parts {
				extractFromWordPart(part, seenPaths, seenVars, &result)
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

	name := args[0]
	name = filepath.Base(name)

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
		if p.Short {
			varName = "$" + p.Param.Value
		} else {
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
		// Recurse into command substitutions
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

func extractHeredoc(redir *syntax.Redirect) *HeredocInfo {
	if redir.Hdoc == nil {
		return nil
	}

	printer := syntax.NewPrinter()
	var delim strings.Builder
	printer.Print(&delim, redir.Word)

	var content strings.Builder
	printer.Print(&content, redir.Hdoc)

	return &HeredocInfo{
		Delimiter: delim.String(),
		Content:   content.String(),
	}
}

func checkPipeToInterpreter(result *ParseResult, binary *syntax.BinaryCmd) {
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
			if interpreters[name] {
				var fullPipe strings.Builder
				printer.Print(&fullPipe, binary)
				result.PipesToInterp = append(result.PipesToInterp, PipeInfo{
					Interpreter: name,
					FullPipe:    fullPipe.String(),
				})
			}
		}
	}
}

func checkRiskyPatterns(result *ParseResult, info *CommandInfo) {
	name := info.Name

	switch name {
	case "rm":
		checkRmPatterns(result, info)
	case "chmod":
		checkChmodPatterns(result, info)
	case "chown":
		checkChownPatterns(result, info)
	case "dd":
		checkDdPatterns(result, info)
	case "mkfs", "mkfs.ext4", "mkfs.xfs", "mkfs.btrfs":
		result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
			Type:    "filesystem_format",
			Detail:  "mkfs command detected",
			Command: info.FullCmd,
		})
	case "mv":
		checkMvPatterns(result, info)
	case "find":
		checkFindPatterns(result, info)
	case "rsync":
		checkRsyncPatterns(result, info)
	case "tar":
		checkTarPatterns(result, info)
	case "cp":
		checkCpPatterns(result, info)
	case "scp":
		checkScpPatterns(result, info)
	case "truncate":
		checkTruncatePatterns(result, info)
	case "ln":
		checkLnPatterns(result, info)
	}
}

func checkRmPatterns(result *ParseResult, info *CommandInfo) {
	hasRecursive := false
	hasForce := false
	dangerousPaths := []string{"/", "/*", "~", "~/*", "$HOME", "$HOME/*", "/Users", "."}
	safePrefixes := []string{"/tmp", "/var/folders", "/private/var/folders"}

	for _, arg := range info.Args {
		if strings.Contains(arg, "-r") || strings.Contains(arg, "-R") {
			hasRecursive = true
		}
		if strings.Contains(arg, "-f") {
			hasForce = true
		}
		for _, dangerous := range dangerousPaths {
			if arg == dangerous || strings.HasPrefix(arg, dangerous+"/") {
				isSafe := false
				for _, safe := range safePrefixes {
					if strings.HasPrefix(arg, safe) {
						isSafe = true
						break
					}
				}
				if !isSafe {
					result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
						Type:    "rm_dangerous_path",
						Detail:  fmt.Sprintf("rm targeting %s", arg),
						Command: info.FullCmd,
					})
					info.IsRisky = true
					info.RiskType = "rm_dangerous_path"
					return
				}
			}
		}
	}

	if hasRecursive && hasForce {
		info.IsRisky = true
		info.RiskType = "rm_rf"
		result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
			Type:    "rm_rf",
			Detail:  "rm -rf detected",
			Command: info.FullCmd,
		})
	}
}

func checkChmodPatterns(result *ParseResult, info *CommandInfo) {
	if len(info.Args) >= 3 {
		for i, arg := range info.Args {
			if arg == "777" && i > 0 {
				if contains(info.Args, "-R") || contains(info.Args, "-r") {
					for _, path := range info.Args[i+1:] {
						if path == "/" || path == "~" || path == "$HOME" {
							result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
								Type:    "chmod_recursive_root",
								Detail:  "recursive chmod 777 on critical path",
								Command: info.FullCmd,
							})
							info.IsRisky = true
							info.RiskType = "chmod_recursive_root"
							return
						}
					}
				}
			}
		}
	}
}

func checkChownPatterns(result *ParseResult, info *CommandInfo) {
	if contains(info.Args, "-R") || contains(info.Args, "-r") {
		for _, arg := range info.Args {
			if arg == "/" || arg == "/*" {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "chown_recursive_root",
					Detail:  "recursive chown on root",
					Command: info.FullCmd,
				})
				info.IsRisky = true
				info.RiskType = "chown_recursive_root"
			}
		}
	}
}

func checkDdPatterns(result *ParseResult, info *CommandInfo) {
	dangerousSources := []string{"/dev/zero", "/dev/random", "/dev/urandom"}
	dangerousTargets := []string{"/dev/sd", "/dev/disk", "/dev/nvme"}

	for _, arg := range info.Args {
		for _, src := range dangerousSources {
			if strings.HasPrefix(arg, "if="+src) {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "dd_dangerous_source",
					Detail:  fmt.Sprintf("dd from %s", src),
					Command: info.FullCmd,
				})
				info.IsRisky = true
				info.RiskType = "dd_dangerous"
			}
		}
		for _, tgt := range dangerousTargets {
			if strings.Contains(arg, "of="+tgt) {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "dd_dangerous_target",
					Detail:  fmt.Sprintf("dd to %s", tgt),
					Command: info.FullCmd,
				})
				info.IsRisky = true
				info.RiskType = "dd_dangerous"
			}
		}
	}
}

func checkMvPatterns(result *ParseResult, info *CommandInfo) {
	if len(info.Args) >= 2 {
		for _, arg := range info.Args[1:] {
			if arg == "/" || arg == "~" || arg == "$HOME" || arg == "." {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "mv_dangerous_path",
					Detail:  fmt.Sprintf("mv targeting %s", arg),
					Command: info.FullCmd,
				})
				info.IsRisky = true
				info.RiskType = "mv_dangerous"
			}
		}
	}
}

func checkFindPatterns(result *ParseResult, info *CommandInfo) {
	for _, arg := range info.Args {
		if arg == "-exec" || arg == "-execdir" || arg == "-ok" || arg == "-okdir" {
			result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
				Type:    "find_exec",
				Detail:  fmt.Sprintf("find with %s can execute arbitrary commands", arg),
				Command: info.FullCmd,
			})
			info.IsRisky = true
			info.RiskType = "find_exec"
			return
		}
		if arg == "-delete" {
			result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
				Type:    "find_delete",
				Detail:  "find with -delete can remove files",
				Command: info.FullCmd,
			})
			info.IsRisky = true
			info.RiskType = "find_delete"
			return
		}
	}
}

func checkRsyncPatterns(result *ParseResult, info *CommandInfo) {
	for _, arg := range info.Args {
		if arg == "--delete" || arg == "--delete-before" || arg == "--delete-after" || arg == "--delete-during" {
			result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
				Type:    "rsync_delete",
				Detail:  "rsync with --delete can remove files at destination",
				Command: info.FullCmd,
			})
			info.IsRisky = true
			info.RiskType = "rsync_delete"
			return
		}
	}
}

func checkTarPatterns(result *ParseResult, info *CommandInfo) {
	hasExtract := false
	dangerousPaths := []string{"/", "/*", "/etc", "/usr", "/bin", "/sbin", "/lib"}
	
	for _, arg := range info.Args {
		if strings.Contains(arg, "x") || arg == "-x" || arg == "--extract" {
			hasExtract = true
		}
	}
	
	if hasExtract {
		for _, arg := range info.Args {
			if arg == "-C" || strings.HasPrefix(arg, "--directory=") {
				continue
			}
			for _, dangerous := range dangerousPaths {
				if arg == dangerous || strings.HasPrefix(arg, dangerous) {
					result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
						Type:    "tar_extract_root",
						Detail:  fmt.Sprintf("tar extracting to %s", arg),
						Command: info.FullCmd,
					})
					info.IsRisky = true
					info.RiskType = "tar_extract_root"
					return
				}
			}
		}
	}
}

func checkCpPatterns(result *ParseResult, info *CommandInfo) {
	dangerousSources := []string{"/dev/null", "/dev/zero"}
	dangerousTargets := []string{"/etc/passwd", "/etc/shadow", "/etc/hosts"}
	
	for _, arg := range info.Args[1:] {
		for _, src := range dangerousSources {
			if arg == src {
				for _, tgt := range info.Args[2:] {
					for _, dangerous := range dangerousTargets {
						if tgt == dangerous {
							result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
								Type:    "cp_overwrite_system",
								Detail:  fmt.Sprintf("cp overwriting %s", tgt),
								Command: info.FullCmd,
							})
							info.IsRisky = true
							info.RiskType = "cp_overwrite_system"
							return
						}
					}
				}
			}
		}
	}
}

func checkScpPatterns(result *ParseResult, info *CommandInfo) {
	dangerousTargets := []string{"/usr/local/bin", "/usr/bin", "/bin", "/sbin", "/etc"}
	
	for _, arg := range info.Args {
		if strings.Contains(arg, ":") {
			for _, dangerous := range dangerousTargets {
				if strings.HasSuffix(arg, dangerous) || strings.HasSuffix(arg, dangerous+"/") {
					result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
						Type:    "scp_to_system",
						Detail:  fmt.Sprintf("scp to system path %s", dangerous),
						Command: info.FullCmd,
					})
					info.IsRisky = true
					info.RiskType = "scp_to_system"
					return
				}
			}
		}
	}
}

func checkTruncatePatterns(result *ParseResult, info *CommandInfo) {
	dangerousFiles := []string{"/etc/passwd", "/etc/shadow", "/etc/hosts", "/etc/fstab"}
	
	for _, arg := range info.Args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		for _, dangerous := range dangerousFiles {
			if arg == dangerous {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "truncate_system",
					Detail:  fmt.Sprintf("truncate on system file %s", arg),
					Command: info.FullCmd,
				})
				info.IsRisky = true
				info.RiskType = "truncate_system"
				return
			}
		}
	}
}

func checkLnPatterns(result *ParseResult, info *CommandInfo) {
	dangerousTargets := []string{"~/.bashrc", "~/.bash_profile", "~/.zshrc", "~/.profile",
		"$HOME/.bashrc", "$HOME/.bash_profile", "$HOME/.zshrc", "$HOME/.profile",
		"/etc/", "/usr/", "/bin/", "/sbin/"}
	
	for _, arg := range info.Args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		for _, dangerous := range dangerousTargets {
			if arg == dangerous || strings.HasPrefix(arg, dangerous) {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "ln_dangerous_target",
					Detail:  fmt.Sprintf("ln targeting %s", arg),
					Command: info.FullCmd,
				})
				info.IsRisky = true
				info.RiskType = "ln_dangerous"
				return
			}
		}
	}
}

func check7zPatterns(result *ParseResult, info *CommandInfo) {
	for _, arg := range info.Args {
		if strings.HasPrefix(arg, "-o/") || arg == "-o/" {
			result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
				Type:    "7z_extract_root",
				Detail:  "7z extracting to root filesystem",
				Command: info.FullCmd,
			})
			info.IsRisky = true
			info.RiskType = "7z_extract_root"
			return
		}
	}
}

func checkGitPatterns(result *ParseResult, info *CommandInfo) {
	if len(info.Args) < 2 {
		return
	}
	if info.Args[1] != "clone" {
		return
	}
	dangerousPaths := []string{"/", "/usr", "/etc", "/bin", "/sbin", "/lib"}
	
	for _, arg := range info.Args[2:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		for _, dangerous := range dangerousPaths {
			if arg == dangerous || strings.HasPrefix(arg, dangerous+"/") {
				result.RiskyPatterns = append(result.RiskyPatterns, RiskyPattern{
					Type:    "git_clone_system",
					Detail:  fmt.Sprintf("git clone to system path %s", arg),
					Command: info.FullCmd,
				})
				info.IsRisky = true
				info.RiskType = "git_clone_system"
				return
			}
		}
	}
}

func extractScriptPath(result *ParseResult, info *CommandInfo, seen map[string]bool) {
	name := info.Name

	if interpreters[name] || name == "source" || name == "." {
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

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s || strings.Contains(item, s) {
			return true
		}
	}
	return false
}

func outputResult(result ParseResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func outputError(msg string) {
	result := ParseResult{Error: msg}
	outputResult(result)
	os.Exit(1)
}
