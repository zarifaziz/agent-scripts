/**
 * Template Registry
 * Manages preview templates for different data structures
 * 
 * To add a new template:
 * 1. Create template-name.js with { name, detect(obj), render(obj) }
 * 2. Create template-name.css with styles scoped to .preview-card .tpl-{name}
 * 3. Register it: TemplateRegistry.register(template)
 */

const TemplateRegistry = {
  templates: [],

  register(template) {
    if (!template.name || !template.detect || !template.render) {
      console.error('Template must have name, detect, and render:', template);
      return;
    }
    this.templates.push(template);
    console.log(`[Templates] Registered: ${template.name}`);
  },

  /**
   * Check if any template matches the content
   * @param {string|object} content - JSON string or parsed object
   * @returns {{ has: boolean, template?: string, reason?: string }}
   */
  check(content) {
    let obj;
    if (typeof content === 'string') {
      try { obj = JSON.parse(content); } catch { return { has: false, reason: 'not-json' }; }
    } else {
      obj = content;
    }

    if (typeof obj !== 'object' || Array.isArray(obj) || obj === null) {
      return { has: false, reason: 'not-object' };
    }

    for (const tpl of this.templates) {
      if (tpl.detect(obj)) {
        return { has: true, template: tpl.name };
      }
    }

    return { has: false, reason: 'no-matching-template' };
  },

  /**
   * Render content using matching template
   * @param {object} obj - Parsed JSON object
   * @returns {string} HTML string or empty if no template matches
   */
  render(obj) {
    if (typeof obj !== 'object' || Array.isArray(obj) || obj === null) {
      return '';
    }

    for (const tpl of this.templates) {
      if (tpl.detect(obj)) {
        try {
          return tpl.render(obj);
        } catch (e) {
          console.error(`[Templates] Render error in ${tpl.name}:`, e);
          return '';
        }
      }
    }

    return '';
  },

  /**
   * List all registered templates
   */
  list() {
    return this.templates.map(t => t.name);
  }
};

// Make globally available
window.TemplateRegistry = TemplateRegistry;
