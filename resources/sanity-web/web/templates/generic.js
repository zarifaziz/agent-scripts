/**
 * Generic Template
 * Fallback for objects with string fields
 * Shows first 6 string key-value pairs
 */

const GenericTemplate = {
  name: 'generic',

  /**
   * Detect if object has string fields (lowest priority fallback)
   */
  detect(obj) {
    const stringKeys = Object.keys(obj).filter(k => typeof obj[k] === 'string');
    return stringKeys.length > 0;
  },

  /**
   * Render generic key-value preview
   */
  render(obj) {
    const stringKeys = Object.keys(obj).filter(k => typeof obj[k] === 'string');
    
    if (!stringKeys.length) return '';

    const rows = stringKeys.slice(0, 6).map(k => `
      <div class="generic-row">
        <div class="generic-key">${escapeHtml(k)}</div>
        <div class="generic-value">${escapeHtml(obj[k])}</div>
      </div>
    `);

    const moreCount = stringKeys.length - 6;
    const moreHint = moreCount > 0 ? `<div class="generic-more">+${moreCount} more fields</div>` : '';

    return `
      <div class="tpl-generic">
        <h4>Preview</h4>
        ${rows.join('')}
        ${moreHint}
      </div>
    `;
  }
};

// Register with registry (should be last - lowest priority)
TemplateRegistry.register(GenericTemplate);
