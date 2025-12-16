#!/bin/bash
# Tests = USAGE.md examples. Backpressure: ✓ on pass, dump on fail.
cd "$(dirname "$0")"

NOTION="./bin/notion"
go build -o bin/notion . 2>/dev/null || { echo "Build failed"; exit 1; }
[[ -f ~/.cache/notion/schema.json ]] || $NOTION schema --refresh > /dev/null 2>&1

# Extract commands
cmds=()
while IFS= read -r line; do
    [[ "$line" == '```bash' ]] && { in_block=true; continue; }
    [[ "$line" == '```' ]] && { in_block=false; continue; }
    [[ "$in_block" == true ]] || continue
    [[ -z "$line" || "$line" =~ ^#.*$ ]] && continue
    line="${line//notion /$NOTION }"
    line="${line//| notion/| $NOTION}"
    cmds+=("$line")
done < USAGE.md

total=${#cmds[@]}
failed=()
tmp=$(mktemp)

for i in "${!cmds[@]}"; do
    cmd="${cmds[$i]}"
    if eval "$cmd" > "$tmp" 2>&1; then
        # Validate JSON for query commands
        if [[ "$cmd" =~ "query" ]] && [[ ! "$cmd" =~ "-o " ]]; then
            jq -e '.pages' "$tmp" > /dev/null 2>&1 || { failed+=("$cmd: bad json"); continue; }
        fi
    else
        failed+=("$cmd: $(head -1 "$tmp")")
    fi
done

rm -f "$tmp"

if [[ ${#failed[@]} -eq 0 ]]; then
    echo "✓ $total tests"
else
    echo "✗ ${#failed[@]}/$total failed"
    printf '  %s\n' "${failed[@]}"
    exit 1
fi
