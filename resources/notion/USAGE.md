# notion CLI

Run `notion schema` first - it shows all properties, types, and valid enum values.

## Introspection

```bash
notion schema
notion users
```

## Query

Pipe JSON to stdin. Shorthand auto-expands to Notion format:

```bash
# Single condition
echo '{"filter":{"status":"done"},"limit":2}' | notion query

# Multiple conditions (AND'd)
echo '{"filter":{"assign":"hemanta","status":"in progress"},"limit":2}' | notion query

# OR conditions
echo '{"filter":{"or":[{"status":"done"},{"status":"in progress"}]},"limit":2}' | notion query

# Full Notion syntax (when shorthand isn't enough)
echo '{"filter":{"property":"Assign","people":{"is_not_empty":true}},"limit":2}' | notion query
```

## Output Formats

```bash
echo '{"filter":{"status":"done"},"limit":2}' | notion query -o table
```

## Single Issue

```bash
notion get ISSUE-123
notion content ISSUE-123
```

## CLI Shorthand

```bash
notion query --status "In progress" --assign hemanta --limit 5
```

## Transformations

All automatic:
- Props: case-insensitive (`status` → `Status`)
- Enums: case-insensitive (`done` → `Done`)
- Users: name → UUID (`hemanta` → `f3687710-...`)
- Aliases: `assign`→`Assign`, `qa`→`QA Engineer`, `title`→`Issue Title`

## Input Schema

```json
{
  "filter": {},
  "sorts": [{"property": "Created At", "direction": "descending"}],
  "limit": 100
}
```
