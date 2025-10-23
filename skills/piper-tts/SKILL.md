---
name: piper-tts
description: Use piper-say TTS tool to provide audio feedback when completing code implementation tasks. Primarily uses English (piper-say) unless user explicitly requests Nepali (piper-sayn).
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Piper TTS Audio Feedback

## Overview
This skill uses the piper-say CLI tool to provide audio feedback to blind developers during code implementation. It announces task completions and progress updates using text-to-speech.

## CLI Paths
- Primary: `piper-say` (English) / `piper-sayn` (Nepali)
- Fallback: `/Users/mac/dev/dotfiles/scripts/piper-say` / `/Users/mac/dev/dotfiles/scripts/piper-sayn`

## Language Selection
- **Default**: Use `piper-say` with English messages
- **When Nepali explicitly requested**: Use `piper-sayn` with Nepali text (converted to Devanagari script)

## When to Use
Provide audio feedback after completing:
- Task implementations
- Bug fixes
- Refactoring work
- Test runs
- Build processes
- Any significant coding milestone

## Usage Patterns

### English (Default)
```bash
piper-say "Task completed successfully"
piper-say "Build finished with no errors"
piper-say "All tests passed"
piper-say "Refactoring complete"
```

### Nepali (When Explicitly Requested)
When user asks for Nepali, convert Roman Nepali to Devanagari script. Keep technical terms in Nepali characters but don't translate them.

```bash
piper-sayn "टास्क कम्प्लिट भयो"
piper-sayn "बिल्ड फिनिश भयो"
piper-sayn "टेस्टहरु पास भए"
```

## Conversion Examples (Nepali Mode Only)

| Roman Nepali | Devanagari |
|--------------|------------|
| Purano resources ko kam sakkyo | पुरानो रिसोर्सेसको काम सक्क्यो |
| Resources ko server down raixa, aru kam sakkyo | रिसोर्सेसको सर्भर डाउन रैछ, अरु काम सक्क्यो |
| Do i also include this refactor? | यो रिफ्याक्टर पनि इन्क्लुड गरुम? |
| Build complete bhayo | बिल्ड कम्प्लिट भयो |
| Tests pass bhae | टेस्टहरु पास भए |

## Implementation Examples

### After Completing a Feature
```bash
# Edit files
# Run tests
# If all successful:
piper-say "Feature implementation complete"
```

### After Fixing a Bug
```bash
# Fix the bug
# Verify the fix
piper-say "Bug fix verified and complete"
```

### After Running Tests
```bash
npm test
if [ $? -eq 0 ]; then
  piper-say "All tests passed successfully"
else
  piper-say "Tests failed, check output"
fi
```

### After Build Process
```bash
npm run build
if [ $? -eq 0 ]; then
  piper-say "Build completed without errors"
else
  piper-say "Build failed, see errors above"
fi
```

### Todo List Completion
When marking final todo as complete:
```bash
# After completing last task
piper-say "All tasks completed"
```

### Nepali Mode Examples (When Explicitly Requested)
```bash
# After completing feature
piper-sayn "फीचर इम्प्लिमेन्टेशन कम्प्लिट भयो"

# After fixing bug
piper-sayn "बग फिक्स कम्प्लिट भयो"

# After tests
piper-sayn "सबै टेस्टहरु पास भए"

# After build
piper-sayn "बिल्ड सक्सेसफुल भयो"
```

## Message Guidelines

### Keep Messages Concise
- One-liners only
- Focus on completion status
- Avoid verbose explanations

Good:
```bash
piper-say "Refactoring complete"
piper-say "Database migration successful"
```

Bad:
```bash
piper-say "I have successfully completed the refactoring of the user authentication module and all related test files"
```

### Use Clear Status Indicators
```bash
piper-say "Task complete"
piper-say "Failed with errors"
piper-say "Ready for review"
piper-say "Tests passing"
piper-say "Build successful"
```

## Volume Control
User can control volume persistently:
```bash
piper-say v 75    # Set to 75%
piper-say v 100   # Set to 100%
```

## Muting
User can mute/unmute:
```bash
piper-say m       # Mute
piper-say u       # Unmute
```

## Integration with Todo Lists
After completing final todo item in a session:
```bash
# Mark last todo as completed
piper-say "All tasks completed"
```

## Key Points
1. **Default to English**: Always use `piper-say` with English unless user explicitly requests Nepali
2. **Concise messages**: One-line status updates only
3. **Call at completion**: Invoke after task finishes, not during
4. **Background execution**: Audio plays async, doesn't block terminal
5. **Nepali conversion**: When using Nepali, convert Roman to Devanagari but keep technical terms phonetically
6. **Status focused**: "Complete", "Done", "Successful", "Failed" etc.

## Common Phrases

### English (Default)
- "Task completed"
- "Tests passing"
- "Build successful"
- "Fix applied"
- "Refactoring done"
- "Ready for review"
- "All tasks finished"
- "Error found, check logs"

### Nepali (When Requested)
- "काम सकियो" (kam sakiyo - task done)
- "टेस्टहरु पास भए" (tests pass bhae)
- "बिल्ड सक्सेसफुल भयो" (build successful bhayo)
- "फिक्स कम्प्लिट भयो" (fix complete bhayo)
- "रिफ्याक्टरिंग सकियो" (refactoring sakiyo)
- "सबै काम सकियो" (sabai kam sakiyo - all work done)
