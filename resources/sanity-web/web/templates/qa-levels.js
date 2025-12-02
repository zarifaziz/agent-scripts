/**
 * QA Levels Template
 * Renders Q&A with difficulty levels:
 * { situation?, easy: { question, answer }, medium: {...}, hard: {...} }
 */

const QALevelsTemplate = {
  name: 'qa-levels',

  /**
   * Detect if object matches QA levels structure
   */
  detect(obj) {
    const levels = ['easy', 'medium', 'hard'].filter(k => 
      obj[k] && typeof obj[k] === 'object'
    );
    return levels.length > 0;
  },

  /**
   * Render QA levels preview HTML
   */
  render(obj) {
    const levels = ['easy', 'medium', 'hard'].filter(k => 
      obj[k] && typeof obj[k] === 'object'
    );

    const parts = [];

    // Situation (if present)
    if (obj.situation) {
      parts.push(`
        <div class="qa-section">
          <div class="qa-label">Situation</div>
          <div class="qa-content">${escapeHtml(obj.situation)}</div>
        </div>
      `);
    }

    // Each difficulty level
    levels.forEach(lvl => {
      const sec = obj[lvl];
      const q = sec.question || '';
      const a = sec.answer || '';
      const diffClass = `diff-${lvl}`;

      parts.push(`
        <div class="qa-section">
          <div class="qa-level-header">
            <span class="qa-level ${diffClass}">${lvl}</span>
          </div>
          <div class="qa-item">
            <span class="qa-prefix">Q:</span>
            <span>${escapeHtml(q)}</span>
          </div>
          <div class="qa-item qa-answer">
            <span class="qa-prefix">A:</span>
            <span>${escapeHtml(a)}</span>
          </div>
        </div>
      `);
    });

    return `
      <div class="tpl-qa-levels">
        <h4>Q&A Preview</h4>
        ${parts.join('')}
      </div>
    `;
  }
};

// Register with registry
TemplateRegistry.register(QALevelsTemplate);
