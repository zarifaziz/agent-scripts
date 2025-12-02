/**
 * Application Template
 * Renders application node output with title and example pairs
 * { title, example1Header, example1Body, example2Header, example2Body, ... }
 */

const ApplicationTemplate = {
  name: 'application',

  /**
   * Detect if object matches application structure
   */
  detect(obj) {
    return obj.title && obj.example1Header && obj.example1Body;
  },

  /**
   * Render application preview HTML
   */
  render(obj) {
    const parts = [];

    // Title first
    if (obj.title) {
      parts.push(`
        <div class="app-title">${escapeHtml(obj.title)}</div>
      `);
    }

    // Find all example pairs (example1, example2, example3, etc.)
    let i = 1;
    while (obj[`example${i}Header`] || obj[`example${i}Body`]) {
      const header = obj[`example${i}Header`] || '';
      const body = obj[`example${i}Body`] || '';
      // Skip reasoning fields
      
      parts.push(`
        <div class="app-example">
          <div class="app-example-header">${escapeHtml(header)}</div>
          <div class="app-example-body">${escapeHtml(body)}</div>
        </div>
      `);
      i++;
    }

    return `
      <div class="tpl-application">
        <h4>Application</h4>
        ${parts.join('')}
      </div>
    `;
  }
};

// Register before generic (higher priority)
TemplateRegistry.register(ApplicationTemplate);
