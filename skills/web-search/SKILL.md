---
name: web-search
description: Search the web
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

search.js and content.js are executable scripts exported in path, just invoke them as below

## Search

```
search.js "query" # Basic search (5 results)
search.js "query" -n 10 # More results
search.js "query" --content # Include page content as markdown
search.js "query" -n 3 --content # Combined
```

## Extract Page Content

```
content.js https://example.com/article
```

Fetches a URL and extracts readable content as markdown.

Output Format

```
--- Result 1 ---
Title: Page Title
Link: https://example.com/page
Snippet: Description from search results
Content: (if --content flag used)
Markdown content extracted from the page...

--- Result 2 ---
...
```

```

```
