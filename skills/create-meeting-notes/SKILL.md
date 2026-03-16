---
description: "Generate meeting notes from Gather recordings. Use when user provides a Gather recording MP4 or asks for meeting notes from a video/recording file."
---

# Create Meeting Notes

Process Gather MP4 recordings into structured, AI-digestible meeting notes with curated screenshots and preserved transcript.

## When to Use

Activate when the user:
- Provides a Gather recording MP4 file path
- Asks for "meeting notes" from a video/recording
- Says something like "process this Gather recording"

## Workflow

### Phase 1: Run the Processing Script

Run the shell wrapper to extract audio, transcribe via Groq Whisper, and extract candidate frames:

```bash
bash ~/.claude/skills/create-meeting-notes/create-meeting-notes "<path-to-mp4>"
```

This produces a `meeting-notes-<basename>/` directory next to the MP4 containing:
- `_candidates/` — candidate frames named `candidate_00m00s.png`
- `transcript.txt` — full Groq Whisper transcript
- `metadata.json` — duration, date, resolution, frame timestamps

If the script fails, report the error to the user and stop.

### Phase 2: Analyze and Curate

1. **Read `metadata.json`** to understand duration and frame timestamps.

2. **Read `transcript.txt`** to understand what was discussed. Identify:
   - Major topic transitions
   - Key decisions or action items
   - Moments where something was shown/demonstrated on screen

3. **View candidate frames** in `_candidates/`. Read them as images to understand visual content. Cross-reference with transcript timestamps to identify which frames correspond to important moments.

4. **Extract details from screenshots** — When frames show code, PRs, docs, or settings:
   - Note specific URLs, file paths, repo names, PR numbers visible on screen
   - Transcribe key code snippets or config values shown
   - Record the exact names of settings, fields, or UI elements being changed
   - These extracted details go in the `## References` section of the notes

5. **Select key frames** based on meeting length:
   - ≤ 15 minutes: 2–5 frames
   - 15–30 minutes: 3–7 frames
   - > 30 minutes: 5–10 frames

   Prioritize frames that show:
   - Unique content (skip duplicates/similar frames)
   - Screen shares, diagrams, or presentations
   - Moments referenced in discussion
   - Key decisions or demonstrations

6. **Create `screenshots/` directory** in the output folder.

7. **Copy selected frames** to `screenshots/` with descriptive names:
   - Format: `1-descriptive-name.png`, `2-descriptive-name.png`, etc.
   - Use `cp` (not move) so we can clean up `_candidates/` safely

8. **Write `notes.md`** using the output format below.

9. **Clean up**:
   - Delete the `_candidates/` directory: `rm -rf <output>/_candidates/`
   - Keep `transcript.txt` — raw timestamped transcript for reference
   - Keep `metadata.json` — lightweight session metadata

### Output Format (notes.md)

```markdown
# Meeting Notes

**Date:** [date from metadata or filename]
**Duration:** [duration from metadata]
**Source:** [original filename]
**Participants:** [names visible in Gather sidebar or mentioned in transcript]

---

## Overview
[2-4 sentence summary combining transcript content + visual context]

## Key Takeaways
- [3-5 bullet points of the most important things discussed/decided]

## References
[URLs, repos, PRs, files, docs, and settings visible on screen or mentioned in discussion]
- **PR:** [URL if visible]
- **Repo:** [repo name if visible]
- **Files discussed:** [file paths seen in code review or file trees]
- **Docs:** [documentation pages visited]
- **Settings changed:** [specific settings/config modified during the meeting]

## Notes

### [Topic Name]
*[timestamp range, e.g., 0:00–3:45]*

[Summary of what was discussed + what was shown on screen]

![1-descriptive-name](screenshots/1-descriptive-name.png)
*[Caption: what this frame shows and why it matters]*

### [Next Topic]
*[timestamp range]*

[Summary...]

![2-descriptive-name](screenshots/2-descriptive-name.png)
*[Caption]*

...

## Action Items
- [ ] [action item with owner if mentioned]
- [ ] [next action item]

---
*Generated from Gather recording via Groq Whisper transcription + visual analysis.*
```

## Important Notes

- The primary consumer of these notes is an AI in a future conversation — optimize for machine parseability and completeness over brevity
- Extract concrete identifiers from screenshots (URLs, file paths, function names, setting names) into text — an AI re-reading these notes may not have image access
- Screenshot captions should explain *why* the frame matters, not just describe it
- If the transcript has speaker labels, attribute quotes and action items
- If a topic has no meaningful visual content, skip the screenshot for that section
- The `transcript.txt` and `metadata.json` are preserved as reference artifacts
