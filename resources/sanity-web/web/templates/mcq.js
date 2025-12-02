/**
 * MCQ Template
 * Renders multiple choice questions with format:
 * { question_1: { question, answerOptions, answer, difficulty }, question_2: ... }
 */

const MCQTemplate = {
  name: 'mcq',

  /**
   * Detect if object matches MCQ structure
   */
  detect(obj) {
    const questionKeys = Object.keys(obj).filter(k => 
      k.startsWith('question_') && obj[k]?.answerOptions
    );
    return questionKeys.length > 0;
  },

  /**
   * Render MCQ preview HTML
   */
  render(obj) {
    const questionKeys = Object.keys(obj)
      .filter(k => k.startsWith('question_') && obj[k]?.answerOptions)
      .sort();

    const parts = questionKeys.map(key => {
      const q = obj[key];
      const qNum = key.replace('question_', '');
      const difficulty = q.difficulty || 'unknown';
      const diffClass = `diff-${difficulty}`;

      // Build options with correct answer highlighted
      const optionsHtml = (q.answerOptions || []).map((opt, i) => {
        const isCorrect = opt === q.answer;
        const letter = String.fromCharCode(65 + i); // A, B, C, D...
        return `
          <div class="mcq-option ${isCorrect ? 'mcq-correct' : ''}">
            <span class="mcq-letter">${letter}</span>
            <span class="mcq-text">${escapeHtml(opt)}</span>
            ${isCorrect ? '<span class="mcq-check">✓</span>' : ''}
          </div>
        `;
      }).join('');

      return `
        <div class="mcq-question">
          <div class="mcq-header">
            <span class="mcq-num">Q${qNum}</span>
            <span class="mcq-diff ${diffClass}">${difficulty}</span>
          </div>
          <div class="mcq-body">${escapeHtml(q.question || '')}</div>
          <div class="mcq-options">${optionsHtml}</div>
          <div class="mcq-answer"><strong>Answer:</strong> ${escapeHtml(q.answer || '')}</div>
        </div>
      `;
    });

    return `
      <div class="tpl-mcq">
        <h4>MCQ Preview</h4>
        ${parts.join('<hr class="mcq-divider"/>')}
      </div>
    `;
  }
};

// Register with registry
TemplateRegistry.register(MCQTemplate);
