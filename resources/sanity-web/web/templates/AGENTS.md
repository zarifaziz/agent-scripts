# Adding Preview Templates

Templates render JSON data as formatted preview cards on hover.

## Create a Template

1. **Create `{name}.js`:**
```js
const MyTemplate = {
  name: 'my-template',
  
  // Return true if this template handles the object
  detect(obj) {
    return obj.someKey && Array.isArray(obj.items);
  },
  
  // Return HTML string (use escapeHtml() for user content)
  render(obj) {
    return `<div class="tpl-my-template">
      <h4>Title</h4>
      <p>${escapeHtml(obj.someKey)}</p>
    </div>`;
  }
};

TemplateRegistry.register(MyTemplate);
```

2. **Create `{name}.css`:** Scope all styles to `.preview-card .tpl-{name}`

3. **Register in `index.html`:**
```html
<link rel="stylesheet" href="/templates/{name}.css">
<script src="/templates/{name}.js"></script>
```

## Rules

- Templates are checked in registration order — put specific templates before generic ones
- `detect()` must be fast — avoid heavy parsing
- `render()` has access to global `escapeHtml(str)` — always escape user content
- CSS vars available: `--accent`, `--error`, `--warn`, `--debug`, `--text`, `--text-2`, `--border`, etc.
- Wrap output in `<div class="tpl-{name}">` for style scoping
- KaTeX is applied automatically to rendered HTML — just output raw `$math$` delimiters

## Testing

Console: `TemplateRegistry.list()` — shows registered templates  
Console: `TemplateRegistry.check(jsonString)` — test detection  
Console: `TemplateRegistry.render(obj)` — test render output
