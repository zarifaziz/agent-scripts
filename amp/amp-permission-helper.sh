#!/usr/bin/env bash
#
# amp-permission-helper - Permission delegate for Amp
#
# Exit codes (per Amp spec):
#   0 = allow
#   1 = ask (Amp's terminal prompt - user can reject)
#   2 = reject (blocked, message to model)
#
# Usage as delegate:
#   Receives AGENT_TOOL_NAME env var and JSON args on stdin
#
# Usage for testing/management:
#   amp-permission-helper --test '{"cmd": "rm -rf /"}'
#   amp-permission-helper --log
#   amp-permission-helper --edit readonly
#   amp-permission-helper --edit sensitive
#   amp-permission-helper --edit reject

set -uo pipefail

# Config
CONFIG_DIR="$HOME/.config/amp-permissions"
LOG_FILE="$CONFIG_DIR/decisions.log"
MAX_LOG_LINES=100

# Ensure config dir exists
mkdir -p "$CONFIG_DIR"

#------------------------------------------------------------------------------
# Config loaders
#------------------------------------------------------------------------------

load_list() {
    local file="$1"
    [[ -f "$file" ]] || return
    grep -v '^#' "$file" | grep -v '^[[:space:]]*$' | sed "s|\$HOME|$HOME|g"
}

load_readonly_commands() {
    load_list "$CONFIG_DIR/readonly-commands.txt"
}

load_sensitive_paths() {
    load_list "$CONFIG_DIR/sensitive-paths.txt"
}

load_reject_patterns() {
    load_list "$CONFIG_DIR/reject-patterns.txt"
}

load_always_allowed_paths() {
    load_list "$CONFIG_DIR/always-allowed-paths.txt"
}

load_always_allowed_commands() {
    load_list "$CONFIG_DIR/always-allowed-commands.txt"
}

load_always_ask_patterns() {
    load_list "$CONFIG_DIR/always-ask-patterns.txt"
}

load_interpreters() {
    load_list "$CONFIG_DIR/interpreters.txt"
}

get_interpreters_regex() {
    load_interpreters | paste -sd'|' -
}

#------------------------------------------------------------------------------
# Logging
#------------------------------------------------------------------------------

log_decision() {
    local decision="$1"
    local tool="$2"
    local detail="$3"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local tmux_ctx=$(get_tmux_context)
    
    echo "[$timestamp] [$tmux_ctx] $decision | $tool | $detail" >> "$LOG_FILE"
    
    # Trim log to max lines
    if [[ -f "$LOG_FILE" ]]; then
        tail -n "$MAX_LOG_LINES" "$LOG_FILE" > "$LOG_FILE.tmp" && mv "$LOG_FILE.tmp" "$LOG_FILE"
    fi
}

#------------------------------------------------------------------------------
# Helpers
#------------------------------------------------------------------------------

get_tmux_context() {
    if [[ -n "${TMUX:-}" ]]; then
        # Use TMUX_PANE to get invoking pane, not currently active one
        if [[ -n "${TMUX_PANE:-}" ]]; then
            tmux display-message -t "$TMUX_PANE" -p '#S:#W(#I)' 2>/dev/null || echo "unknown"
        else
            tmux display-message -p '#S:#W(#I)' 2>/dev/null || echo "unknown"
        fi
    else
        echo "no-tmux"
    fi
}

resolve_path() {
    local path="$1"
    [[ -z "$path" ]] && return
    path="${path/#\~/$HOME}"
    path="${path/\$HOME/$HOME}"
    if [[ -e "$path" ]]; then
        realpath "$path" 2>/dev/null || echo "$path"
    else
        local dir=$(dirname "$path")
        local base=$(basename "$path")
        if [[ -e "$dir" ]]; then
            echo "$(realpath "$dir" 2>/dev/null)/$base"
        else
            echo "$path"
        fi
    fi
}

is_under_pwd() {
    local path="$1"
    local resolved=$(resolve_path "$path")
    local pwd_resolved=$(resolve_path "$WORK_DIR")
    [[ "$resolved" == "$pwd_resolved"* ]]
}

extract_paths() {
    local cmd="$1"
    local raw=$(echo "$cmd" | grep -oE '(^|[[:space:]])(~[^[:space:]]*|/[^[:space:];&|<>]*|\$HOME[^[:space:]]*)' | \
        sed 's/^[[:space:]]*//' | grep -v '^\.\.\?/' || true)
    [[ -z "$raw" ]] && return
    while IFS= read -r p || [[ -n "$p" ]]; do
        [[ -n "$p" ]] && resolve_path "$p"
    done <<< "$raw"
}

prompt_user() {
    local title="$1"
    local message="$2"
    
    command -v osascript &>/dev/null || return 1
    
    message="${message//\"/\\\"}"
    title="${title//\"/\\\"}"
    
    while true; do
        local osa_output result gave_up
        local buttons='{"Deny", "Allow"}'
        
        # Add Visit button if in tmux
        if [[ -n "${TMUX:-}" ]]; then
            buttons='{"Deny", "Visit", "Allow"}'
        fi
        
        # Get both button result and gave up status
        osa_output=$(osascript -e "
            set dialogResult to display dialog \"$message\" with title \"$title\" buttons $buttons default button \"Deny\" with icon caution giving up after 60
            return (button returned of dialogResult) & \"|\" & (gave up of dialogResult)
        " 2>/dev/null) || osa_output="|true"
        
        result="${osa_output%%|*}"
        gave_up="${osa_output##*|}"
        
        # Handle Visit button - switch to tmux session/window and re-prompt
        if [[ "$result" == "Visit" ]]; then
            local target
            if [[ -n "${TMUX_PANE:-}" ]]; then
                target=$(tmux display-message -t "$TMUX_PANE" -p '#S:#I' 2>/dev/null)
            else
                target=$(tmux display-message -p '#S:#I' 2>/dev/null)
            fi
            # Switch to target window (works across sessions too)
            tmux switch-client -t "$target" 2>/dev/null || \
            tmux select-window -t "$target" 2>/dev/null
            # Bring terminal app to front
            local term_app
            if pgrep -qf "WezTerm"; then
                term_app="WezTerm"
            elif pgrep -qf "iTerm"; then
                term_app="iTerm"
            elif pgrep -qf "Alacritty"; then
                term_app="Alacritty"
            elif pgrep -qf "kitty"; then
                term_app="kitty"
            elif pgrep -qf "Terminal.app"; then
                term_app="Terminal"
            fi
            [[ -n "$term_app" ]] && open -a "$term_app"
            sleep 0.3  # Brief pause to let switch complete
            continue  # Re-show dialog
        fi
        
        # Send notification on timeout
        if [[ "$gave_up" == "true" ]]; then
            local ctx notif_title
            if [[ -n "${TMUX:-}" ]]; then
                ctx=$(tmux display-message -p '#S:#W(#I)' 2>/dev/null)
                notif_title="Amp Permission [$ctx]"
            else
                notif_title="Amp Permission"
            fi
            local pwd_short=$(echo "$WORK_DIR" | awk -F/ '{print $(NF-1)"/"$NF}')
            osascript -e "display notification \"Timed out - auto denied\n[$pwd_short]\" with title \"$notif_title\" sound name \"Basso\"" 2>/dev/null
            log_decision "TIMEOUT" "prompt" "$title: $message"
        fi
        
        break
    done
    
    [[ "$result" == "Allow" ]]
}

#------------------------------------------------------------------------------
# Checks
#------------------------------------------------------------------------------

is_always_allowed_path() {
    local path="$1"
    local resolved=$(resolve_path "$path")
    while IFS= read -r allowed || [[ -n "$allowed" ]]; do
        [[ -z "$allowed" ]] && continue
        if [[ "$resolved" == "$allowed"* ]]; then
            return 0
        fi
    done <<< "$(load_always_allowed_paths)"
    return 1
}

load_all_safe_commands() {
    # Combine readonly + always-allowed commands
    load_readonly_commands
    load_always_allowed_commands
}

matches_always_ask_pattern() {
    local cmd="$1"
    while IFS= read -r pattern || [[ -n "$pattern" ]]; do
        [[ -z "$pattern" ]] && continue
        if [[ "$cmd" =~ $pattern ]]; then
            echo "$pattern"
            return 0
        fi
    done <<< "$(load_always_ask_patterns)"
    return 1
}

is_stdin_to_interpreter() {
    local cmd="$1"
    local interp_regex=$(get_interpreters_regex)
    
    # Detect: | bash, | sh, | python, etc.
    if [[ "$cmd" =~ \|[[:space:]]*($interp_regex)([[:space:]]|$) ]]; then
        echo "pipe to ${BASH_REMATCH[1]}"
        return 0
    elif [[ "$cmd" =~ \<\<[[:space:]]*[\'\"]?[A-Za-z_]+[\'\"]? ]]; then
        echo "heredoc"
        return 0
    elif [[ "$cmd" =~ \<\<\<[[:space:]]* ]]; then
        echo "herestring"
        return 0
    fi
    return 1
}

extract_script_path() {
    local cmd="$1"
    local interp_regex=$(get_interpreters_regex)
    
    if [[ "$cmd" =~ (^|[;\&\|[:space:]])($interp_regex|source)[[:space:]]+([^[:space:]\;\&\|]+) ]]; then
        echo "${BASH_REMATCH[3]}"
    elif [[ "$cmd" =~ (^|[;\&\|[:space:]])\.[[:space:]]+([^[:space:]\;\&\|]+) ]]; then
        echo "${BASH_REMATCH[2]}"
    elif [[ "$cmd" =~ (^|[;\&\|[:space:]])(\.\/[^[:space:]\;\&\|]+) ]]; then
        echo "${BASH_REMATCH[2]}"
    fi
}

scan_script_file() {
    local cmd="$1"
    local depth="${2:-0}"
    local visited="${3:-}"
    local max_depth=3
    
    # Depth limit - flag as risky if we can't scan deep enough
    if [[ $depth -ge $max_depth ]]; then
        echo "ASK:max depth ($max_depth) reached, can't fully scan"
        return 0
    fi
    
    # Extract and resolve script path
    local script_path=$(extract_script_path "$cmd")
    [[ -z "$script_path" ]] && return 1
    script_path=$(resolve_path "$script_path")
    [[ ! -f "$script_path" ]] && return 1
    
    # Circular reference check
    [[ "$visited" == *"|$script_path|"* ]] && return 1
    visited="$visited|$script_path|"
    
    local content=$(cat "$script_path" 2>/dev/null)
    [[ -z "$content" ]] && return 1
    
    # Scan for reject patterns - these are BLOCK level (catastrophic)
    while IFS= read -r pattern || [[ -n "$pattern" ]]; do
        [[ -z "$pattern" ]] && continue
        if [[ "$content" == *"$pattern"* ]]; then
            echo "BLOCK:script:$script_path contains '$pattern'"
            return 0
        fi
    done <<< "$(load_reject_patterns)"
    
    # Scan for always-ask patterns (strip ^ anchor for file scanning) - ASK level
    while IFS= read -r pattern || [[ -n "$pattern" ]]; do
        [[ -z "$pattern" ]] && continue
        local file_pattern="${pattern#^}"
        if [[ "$content" =~ $file_pattern ]]; then
            echo "ASK:script:$script_path matches '$pattern'"
            return 0
        fi
    done <<< "$(load_always_ask_patterns)"
    
    # Recursive: scan for nested script executions
    local nested_result
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
        if nested_result=$(scan_script_file "$line" $((depth + 1)) "$visited"); then
            echo "$nested_result"
            return 0
        fi
    done <<< "$content"
    
    return 1
}

all_paths_always_allowed() {
    local paths="$1"
    [[ -z "$paths" ]] && return 1
    while IFS= read -r p || [[ -n "$p" ]]; do
        [[ -z "$p" ]] && continue
        if ! is_always_allowed_path "$p"; then
            return 1
        fi
    done <<< "$paths"
    return 0
}

is_rejected_pattern() {
    local cmd="$1"
    local expanded_cmd="${cmd//\$HOME/$HOME}"
    expanded_cmd="${expanded_cmd//\~/$HOME}"
    while IFS= read -r pattern || [[ -n "$pattern" ]]; do
        [[ -z "$pattern" ]] && continue
        if [[ "$cmd" == *"$pattern"* ]] || [[ "$expanded_cmd" == *"$pattern"* ]]; then
            return 0
        fi
    done <<< "$(load_reject_patterns)"
    return 1
}

is_safe_command() {
    local cmd="$1"
    
    # Reject if shell metacharacters present (prevent piggybacking)
    if [[ "$cmd" =~ [\|\;\&\$\`\(\)\>] ]] || [[ "$cmd" == *'<('* ]]; then
        return 1
    fi
    
    local first_word="${cmd%% *}"
    first_word="${first_word##*/}"
    
    while IFS= read -r safe_cmd || [[ -n "$safe_cmd" ]]; do
        [[ -z "$safe_cmd" ]] && continue
        if [[ "$first_word" == "$safe_cmd" ]]; then
            # sed -i is a write
            if [[ "$safe_cmd" == "sed" ]] && [[ "$cmd" =~ sed[[:space:]]+-i ]]; then
                return 1
            fi
            return 0
        fi
    done <<< "$(load_all_safe_commands)"
    return 1
}

matches_sensitive_path() {
    local path="$1"
    local resolved=$(resolve_path "$path")
    
    while IFS= read -r sensitive || [[ -n "$sensitive" ]]; do
        [[ -z "$sensitive" ]] && continue
        local sensitive_resolved=$(resolve_path "$sensitive")
        if [[ "$resolved" == "$sensitive_resolved"* ]]; then
            echo "$sensitive"
            return 0
        fi
    done <<< "$(load_sensitive_paths)"
    return 1
}

#------------------------------------------------------------------------------
# Tool handlers
#------------------------------------------------------------------------------

handle_bash() {
    local cmd=$(echo "$ARGS" | jq -r '.cmd // empty' 2>/dev/null)
    [[ -z "$cmd" ]] && { log_decision "ALLOW" "Bash" "empty command"; exit 0; }
    
    # 1. Catastrophic patterns - always block
    if is_rejected_pattern "$cmd"; then
        log_decision "BLOCK" "Bash" "$cmd"
        echo "BLOCKED: Catastrophic command pattern" >&2
        exit 2
    fi
    
    # 1a. Scan script files for risky content (BLOCK or ASK)
    local script_risk
    if script_risk=$(scan_script_file "$cmd"); then
        # Check if it's a BLOCK (catastrophic) or ASK level risk
        if [[ "$script_risk" == BLOCK:* ]]; then
            local reason="${script_risk#BLOCK:}"
            log_decision "BLOCK" "Bash" "catastrophic pattern in script: $reason | $cmd"
            echo "BLOCKED: $reason" >&2
            exit 2
        else
            # ASK level - prompt user
            local reason="${script_risk#ASK:}"
            local tmux_ctx=$(get_tmux_context)
            if prompt_user "[$tmux_ctx] SCRIPT RISK" "$reason\n\nCmd: $cmd"; then
                log_decision "ALLOW" "Bash" "user approved risky script: $cmd"
                exit 0
            else
                log_decision "DENY" "Bash" "user denied risky script: $cmd"
                exit 1
            fi
        fi
    fi
    
    # 1b. Stdin to interpreter (heredoc, pipe to bash/python, etc.)
    local stdin_risk
    if stdin_risk=$(is_stdin_to_interpreter "$cmd"); then
        local tmux_ctx=$(get_tmux_context)
        if prompt_user "[$tmux_ctx] STDIN EXEC" "$stdin_risk\n\nCmd: $cmd"; then
            log_decision "ALLOW" "Bash" "user approved stdin exec: $cmd"
            exit 0
        else
            log_decision "DENY" "Bash" "user denied stdin exec: $cmd"
            exit 1
        fi
    fi
    
    # 2. Safe commands - but check sensitive paths first
    if is_safe_command "$cmd"; then
        local paths=$(extract_paths "$cmd")
        if [[ -n "$paths" ]]; then
            while IFS= read -r p || [[ -n "$p" ]]; do
                [[ -z "$p" ]] && continue
                local sensitive_match
                if sensitive_match=$(matches_sensitive_path "$p"); then
                    local tmux_ctx=$(get_tmux_context)
                    if prompt_user "[$tmux_ctx] SENSITIVE" "$p\n\nLocation: $sensitive_match\n\nCmd: $cmd"; then
                        log_decision "ALLOW" "Bash" "user approved sensitive safe cmd: $cmd"
                        exit 0
                    else
                        log_decision "DENY" "Bash" "user denied sensitive safe cmd: $cmd"
                        exit 1
                    fi
                fi
            done <<< "$paths"
        fi
        log_decision "ALLOW" "Bash" "safe cmd: $cmd"
        exit 0
    fi
    
    # 3. Always-ask patterns (brew, git reset, etc) - always prompt
    local ask_pattern
    if ask_pattern=$(matches_always_ask_pattern "$cmd"); then
        local tmux_ctx=$(get_tmux_context)
        if prompt_user "[$tmux_ctx] CONFIRM" "$cmd"; then
            log_decision "ALLOW" "Bash" "user approved ($ask_pattern): $cmd"
            exit 0
        else
            log_decision "DENY" "Bash" "user denied ($ask_pattern): $cmd"
            exit 1
        fi
    fi
    
    # 3. Extract paths and check always-allowed dirs first
    local paths=$(extract_paths "$cmd")
    
    # 3a. All paths in always-allowed dirs (/tmp, /var/folders) - allow
    if all_paths_always_allowed "$paths"; then
        log_decision "ALLOW" "Bash" "always-allowed path: $cmd"
        exit 0
    fi
    
    # 4. No paths or all under PWD - allow
    if [[ -z "$paths" ]]; then
        log_decision "ALLOW" "Bash" "no paths: $cmd"
        exit 0
    fi
    
    local outside_paths=()
    while IFS= read -r p || [[ -n "$p" ]]; do
        [[ -z "$p" ]] && continue
        # Skip if under PWD or always-allowed
        if ! is_under_pwd "$p" && ! is_always_allowed_path "$p"; then
            outside_paths+=("$p")
        fi
    done <<< "$paths"
    
    if [[ ${#outside_paths[@]} -eq 0 ]]; then
        log_decision "ALLOW" "Bash" "all paths in PWD: $cmd"
        exit 0
    fi
    
    # 5. Check sensitive paths
    for p in "${outside_paths[@]}"; do
        local sensitive_match
        if sensitive_match=$(matches_sensitive_path "$p"); then
            local tmux_ctx=$(get_tmux_context)
            if prompt_user "[$tmux_ctx] SENSITIVE" "$p\n\nLocation: $sensitive_match\n\nCmd: $cmd"; then
                log_decision "ALLOW" "Bash" "user approved sensitive: $cmd"
                exit 0
            else
                log_decision "DENY" "Bash" "user denied sensitive: $cmd"
                exit 1
            fi
        fi
    done
    
    # 6. Outside PWD - prompt
    local tmux_ctx=$(get_tmux_context)
    if prompt_user "[$tmux_ctx] OUTSIDE PWD" "Paths: ${outside_paths[*]}\n\nPWD: $WORK_DIR\n\nCmd: $cmd"; then
        log_decision "ALLOW" "Bash" "user approved outside: $cmd"
        exit 0
    else
        log_decision "DENY" "Bash" "user denied outside: $cmd"
        exit 1
    fi
}

handle_file_tool() {
    local path=$(echo "$ARGS" | jq -r '.path // empty' 2>/dev/null)
    [[ -z "$path" ]] && { log_decision "ALLOW" "$TOOL_NAME" "no path"; exit 0; }
    
    # Always-allowed paths (/tmp, /var/folders) - allow
    if is_always_allowed_path "$path"; then
        log_decision "ALLOW" "$TOOL_NAME" "always-allowed: $path"
        exit 0
    fi
    
    # Under PWD - allow
    if is_under_pwd "$path"; then
        log_decision "ALLOW" "$TOOL_NAME" "in PWD: $path"
        exit 0
    fi
    
    # Sensitive path
    local sensitive_match
    if sensitive_match=$(matches_sensitive_path "$path"); then
        local tmux_ctx=$(get_tmux_context)
        if prompt_user "[$tmux_ctx] SENSITIVE FILE" "$path\n\nLocation: $sensitive_match"; then
            log_decision "ALLOW" "$TOOL_NAME" "user approved sensitive: $path"
            exit 0
        else
            log_decision "DENY" "$TOOL_NAME" "user denied sensitive: $path"
            exit 1
        fi
    fi
    
    # Outside PWD
    local tmux_ctx=$(get_tmux_context)
    if prompt_user "[$tmux_ctx] OUTSIDE PWD" "$path\n\nPWD: $WORK_DIR"; then
        log_decision "ALLOW" "$TOOL_NAME" "user approved outside: $path"
        exit 0
    else
        log_decision "DENY" "$TOOL_NAME" "user denied outside: $path"
        exit 1
    fi
}

#------------------------------------------------------------------------------
# CLI commands
#------------------------------------------------------------------------------

cmd_test() {
    local json="$1"
    local tool="${2:-Bash}"
    
    TOOL_NAME="$tool"
    ARGS="$json"
    WORK_DIR="${PWD:-$(pwd)}"
    
    echo "Testing: $tool"
    echo "Args: $json"
    echo "PWD: $WORK_DIR"
    echo "---"
    
    case "$tool" in
        Bash) handle_bash ;;
        edit_file|create_file) handle_file_tool ;;
        *) echo "Unknown tool: $tool"; exit 1 ;;
    esac
}

cmd_log() {
    if [[ -f "$LOG_FILE" ]]; then
        cat "$LOG_FILE"
    else
        echo "No log entries yet"
    fi
}

cmd_edit() {
    local which="$1"
    local file
    case "$which" in
        readonly) file="$CONFIG_DIR/readonly-commands.txt" ;;
        sensitive) file="$CONFIG_DIR/sensitive-paths.txt" ;;
        reject) file="$CONFIG_DIR/reject-patterns.txt" ;;
        paths) file="$CONFIG_DIR/always-allowed-paths.txt" ;;
        commands) file="$CONFIG_DIR/always-allowed-commands.txt" ;;
        ask) file="$CONFIG_DIR/always-ask-patterns.txt" ;;
        interpreters) file="$CONFIG_DIR/interpreters.txt" ;;
        *) echo "Unknown config: $which (use: readonly, sensitive, reject, paths, commands, ask, interpreters)"; exit 1 ;;
    esac
    ${EDITOR:-vim} "$file"
}

cmd_help() {
    cat <<'EOF'
amp-permission-helper - Amp permission delegate

DELEGATE MODE (called by Amp):
  Receives AGENT_TOOL_NAME env var and JSON args on stdin

CLI MODE:
  --test JSON [TOOL]  Test a command without executing
  --log               Show decision log
  --edit CONFIG       Edit config (readonly, sensitive, reject)
  --help              Show this help

CONFIG FILES (~/.config/amp-permissions/):
  readonly-commands.txt  - Commands allowed without prompt
  sensitive-paths.txt    - Paths that always prompt
  reject-patterns.txt    - Patterns that are always blocked

EXAMPLES:
  amp-permission-helper --test '{"cmd": "cat /etc/hosts"}'
  amp-permission-helper --test '{"cmd": "rm -rf ~/"}' Bash
  amp-permission-helper --test '{"path": "/etc/hosts"}' edit_file
  amp-permission-helper --log
  amp-permission-helper --edit readonly
EOF
}

#------------------------------------------------------------------------------
# Main
#------------------------------------------------------------------------------

# CLI mode
if [[ "${1:-}" == --* ]]; then
    case "$1" in
        --test) cmd_test "${2:-'{}'}" "${3:-Bash}" ;;
        --log) cmd_log ;;
        --edit) cmd_edit "${2:-}" ;;
        --help|-h) cmd_help ;;
        *) echo "Unknown option: $1"; cmd_help; exit 1 ;;
    esac
    exit $?
fi

# Delegate mode
TOOL_NAME="${AGENT_TOOL_NAME:-}"
ARGS=$(cat)
WORK_DIR="${PWD:-$(pwd)}"

case "$TOOL_NAME" in
    Bash) handle_bash ;;
    edit_file|create_file) handle_file_tool ;;
    *) exit 0 ;;  # Allow unknown tools
esac
