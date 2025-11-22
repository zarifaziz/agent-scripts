const app = document.getElementById('app');
let socket;
const records = [];
const renderedIndices = new Set();
const openLogIndices = []; // Track indices of open logs for rolling window
const isPinView = location.pathname.startsWith('/p/');
const pinId = isPinView ? location.pathname.split('/p/')[1]?.split('/')[0] : '';

// Persistence Helpers
function getToggleState(key, defaultState) {
  const stored = localStorage.getItem(key);
  return stored === null ? defaultState : stored === 'true';
}

function setToggleState(key, state) {
  localStorage.setItem(key, String(state));
  // Dispatch sync event
  window.dispatchEvent(new CustomEvent('toggle-sync', { detail: { key, state } }));
}

// Live Sync Listener
window.addEventListener('toggle-sync', (e) => {
  const { key, state } = e.detail;
  document.querySelectorAll(`details[data-persistence-key="${key}"]`).forEach(el => {
    if (el.open !== state) el.open = state;
  });
});

// Cross-tab sync
window.addEventListener('storage', (e) => {
  if (e.key) {
    const state = e.newValue === 'true';
    window.dispatchEvent(new CustomEvent('toggle-sync', { detail: { key: e.key, state } }));
  }
});

async function fetchStreamText(url = '/stream') {
  const res = await fetch(url);
  return await res.text();
}

function connect() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  socket = new WebSocket(`${proto}//${location.host}/ws`);
  socket.onmessage = (ev) => {
    ev.data.split('\n').forEach(line => {
      if (!line.trim()) return;
      try {
        const record = JSON.parse(line);
        records.push(record);
        renderIncremental(record);
      } catch { }
    });
  };
  socket.onclose = () => setTimeout(connect, 1000);
}

async function loadPinAndRender() {
  const parts = location.pathname.split('/p/');
  const id = parts[1] || '';
  if (!id) return;
  const text = await fetchStreamText(`/pin/${id}`);
  text.split('\n').forEach(line => {
    if (!line.trim()) return;
    try {
      const r = JSON.parse(line);
      records.push(r);
    } catch { }
  });
  // render all accumulated
  records.forEach(renderIncremental);
}

if (isPinView) {
  loadPinAndRender();
} else {
  connect();
}

function renderIncremental(r) {
  const key = r.index ?? r._idx ?? Math.random();

  // If it's a block or text, we might need to find the parent log if we want to append to it
  // But the current structure groups by index. 
  // If we receive a log type, we create a card.
  // If we receive a block type, we append to existing card.

  if (r.type === 'log' || r.type === 'text') {
    if (renderedIndices.has(key)) return; // Already rendered
    renderedIndices.add(key);

    const log = r.type === 'text'
      ? { summary: { message: r.text, severity: 'INFO' }, data: {}, json: {}, index: key }
      : r;

    const card = createLogCard(log);
    app.appendChild(card);

    // Rolling Open Logic
    // Open the new card
    const details = card.querySelector('details.card'); // createCardBase returns details directly now? No, createCardBase returns details.card
    if (details) {
      details.open = true;
      openLogIndices.push(key);

      // Keep only last 3 open
      if (openLogIndices.length > 3) {
        const toCloseIdx = openLogIndices.shift();
        const toCloseCard = document.getElementById(`log-${toCloseIdx}`);
        if (toCloseCard) toCloseCard.open = false;
      }
    }

    // Scroll to bottom? Maybe user wants that, but not explicitly requested. 
    // Standard terminal behavior usually scrolls.
    // window.scrollTo(0, document.body.scrollHeight);
  } else if (r.type === 'block') {
    // Find the card for this block
    const card = document.getElementById(`log-${key}`);
    if (card) {
      const content = card.querySelector('.content');
      // Append block. 
      // We need to find where to append. 
      // For prompt logs, blocks usually go before Output? Or just at end of content?
      // In previous renderPromptLog, blocks were appended near end.
      // Let's just append to content for now.
      content.appendChild(createCollapsibleSection(`Block: ${r.block.path}`, r.block.value, `block:${r.block.path}`, false));
    }
  }
}

function createLogCard(log) {
  const msg = log.summary.message || '';
  let card;
  if (msg.includes('[prompt-input-and-output]')) {
    card = renderPromptLog(log);
  } else if (msg.includes('[latex-correction]')) {
    card = renderLatexLog(log);
  } else {
    card = renderGenericLog(log);
  }
  // Set ID for updates
  const key = log.index ?? log._idx;
  if (key) card.id = `log-${key}`;
  return card;
}

function renderPromptLog(log) {
  const card = createCardBase(log, 'log-prompt');
  const content = card.querySelector('.content');

  // 1. Metadata Grid (OTel + Provider + Additional)
  const grid = document.createElement('div');
  grid.className = 'grid';

  // OTel
  if (log.otel) grid.appendChild(makeSmartTable('OTel Context', log.otel, 'log-prompt:otel', false));

  // Provider (only if not empty)
  if (log.provider && Object.keys(log.provider).length > 0) {
    grid.appendChild(makeSmartTable('Provider Config', log.provider, 'log-prompt:provider', false));
  }

  // Additional Data (Remaining)
  const remainingData = { ...log.data };
  ['system', 'user', 'output', 'rawOutput', 'nodeType', 'promptType'].forEach(k => delete remainingData[k]);

  if (Object.keys(remainingData).length > 0) {
    grid.appendChild(makeSmartTable('Additional Data', remainingData, 'log-prompt:data', false));
  }

  content.appendChild(grid);

  // 2. Main Content (System, User, Raw)
  const contentSection = document.createElement('div');

  if (log.data.system) contentSection.appendChild(createCollapsibleSection('System Prompt', log.data.system, 'log-prompt:system', false));
  if (log.data.user) contentSection.appendChild(createCollapsibleSection('User Prompt', log.data.user, 'log-prompt:user', false));
  if (log.data.rawOutput) contentSection.appendChild(createCollapsibleSection('Raw Output', log.data.rawOutput, 'log-prompt:raw', false));

  content.appendChild(contentSection);

  // 3. Output (Expanded and Last)
  if (log.data.output) {
    content.appendChild(createCollapsibleSection('Output', log.data.output, 'log-prompt:output', true));
  }

  return card;
}

function renderLatexLog(log) {
  const card = createCardBase(log, 'log-latex');
  const content = card.querySelector('.content');

  const grid = document.createElement('div');
  grid.className = 'grid';
  grid.appendChild(makeSmartTable('OTel', log.otel, 'log-latex:otel', true));
  const dataTable = makeSmartTable('Correction Data', log.data, 'log-latex:data', true);
  dataTable.style.gridColumn = '1 / -1';
  grid.appendChild(dataTable);
  content.appendChild(grid);

  return card;
}

function renderGenericLog(log) {
  const card = createCardBase(log, '');
  const content = card.querySelector('.content');

  const grid = document.createElement('div');
  grid.className = 'grid';
  if (log.otel && Object.keys(log.otel).length) grid.appendChild(makeSmartTable('OTel', log.otel, 'generic:otel', true));
  if (log.provider && Object.keys(log.provider).length) grid.appendChild(makeSmartTable('Provider', log.provider, 'generic:provider', true));

  if (log.data && Object.keys(log.data).length) {
    const dataTable = makeSmartTable('Data', log.data, 'generic:data', true);
    if (log.otel && log.provider) {
      dataTable.style.gridColumn = '1 / -1';
    }
    grid.appendChild(dataTable);
  }
  content.appendChild(grid);

  return card;
}

function createCardBase(log, className) {
  const details = document.createElement('details');
  details.className = `card ${className}`;

  const s = log.summary || {};
  const severity = (s.severity || 'INFO').toUpperCase();
  const time = s.time ? new Date(s.time).toLocaleTimeString() : '';

  const summary = document.createElement('summary');
  summary.className = 'row';
  summary.tabIndex = 0;

  // ID Icons
  const icons = document.createElement('div');
  icons.className = 'id-icons';

  const findId = (key) => (log.otel && log.otel[key]) || (log.data && log.data[key]) || (log.provider && log.provider[key]);

  const lpId = findId('lessonPlanID');
  const traceId = findId('traceID');
  const reqId = findId('requestID');
  const userId = findId('userID') || findId('authorizedUserID');

  if (lpId) icons.appendChild(createIdIcon('lp', lpId));
  if (traceId) icons.appendChild(createIdIcon('T', traceId));
  if (reqId) icons.appendChild(createIdIcon('R', reqId));
  if (userId) icons.appendChild(createIdIcon('U', userId));

  // Message Processing
  let messageHtml = '';
  const rawMsg = s.message || '(no message)';

  // Extract [prefix]
  const prefixMatch = rawMsg.match(/^\[([\w-]+)\]/);
  let cleanMsg = rawMsg;
  let prefixLabel = '';

  if (prefixMatch) {
    prefixLabel = `<span class="log-prefix">${prefixMatch[1]}</span>`;
    cleanMsg = rawMsg.replace(prefixMatch[0], '').trim();
  }

  // Prompt/Node Type Labels
  let extraLabels = '';
  const promptType = (log.data && log.data.promptType) || (log.provider && log.provider.promptType);
  const nodeType = (log.data && log.data.nodeType) || (log.provider && log.provider.nodeType);

  if (promptType) extraLabels += `<span class="prompt-type">${promptType}</span>`;
  if (nodeType) extraLabels += `<span class="node-type">${nodeType}</span>`;

  messageHtml = `
      <div class="prompt-preview">
        ${prefixLabel}
        ${extraLabels}
        <span>${escapeHtml(cleanMsg)}</span>
      </div>
    `;

  summary.innerHTML = `
      <span class="time">${time}</span>
      <span class="badge ${severity}">${severity}</span>
      <div class="message">${messageHtml}</div>
    `;

  const pill = document.createElement('span');
  pill.className = 'pill';
  pill.textContent = `caller ${(log.otel || {}).caller || s.caller || ''}`;

  summary.appendChild(icons);
  summary.appendChild(pill);

  const content = document.createElement('div');
  content.className = 'content';

  details.appendChild(summary);
  details.appendChild(content);

  return details;
}

function createIdIcon(label, value) {
  const span = document.createElement('span');
  span.className = 'id-icon';
  span.textContent = label;
  span.title = `Copy ${value}`;
  span.onclick = (e) => {
    e.preventDefault();
    e.stopPropagation();
    navigator.clipboard.writeText(value);
    const original = span.textContent;
    span.textContent = '✓';
    setTimeout(() => span.textContent = original, 1000);
  };
  return span;
}

function createCollapsibleSection(title, content, persistenceKey, defaultOpen) {
  const details = document.createElement('details');
  details.className = 'nested';
  details.dataset.persistenceKey = persistenceKey; // For Live Sync

  // Persistence
  const isOpen = getToggleState(persistenceKey, defaultOpen);
  details.open = isOpen;
  details.ontoggle = () => setToggleState(persistenceKey, details.open);

  const summary = document.createElement('summary');
  summary.textContent = title;
  details.appendChild(summary);

  const wrapper = document.createElement('div');
  wrapper.className = 'code-block-wrapper';

  const pre = document.createElement('pre');
  const textContent = typeof content === 'object' ? JSON.stringify(content, null, 2) : content;
  pre.textContent = textContent;

  const copyBtn = document.createElement('button');
  copyBtn.className = 'copy-btn';
  copyBtn.textContent = 'Copy';
  copyBtn.onclick = (e) => {
    e.stopPropagation();
    navigator.clipboard.writeText(textContent);
    copyBtn.textContent = 'Copied!';
    setTimeout(() => copyBtn.textContent = 'Copy', 1500);
  };

  // Long Press to Toggle
  let pressTimer;
  wrapper.onmousedown = (e) => {
    if (e.target === copyBtn) return;
    pressTimer = setTimeout(() => {
      details.open = !details.open;
    }, 500); // 500ms long press
  };
  wrapper.onmouseup = () => clearTimeout(pressTimer);
  wrapper.onmouseleave = () => clearTimeout(pressTimer);

  wrapper.appendChild(pre);
  wrapper.appendChild(copyBtn);
  details.appendChild(wrapper);

  return details;
}

function makeSmartTable(title, obj, persistenceKey, defaultOpen) {
  const wrap = document.createElement('div');
  wrap.className = 'table-wrap';

  // Use details for the table itself
  const details = document.createElement('details');
  details.className = 'nested';
  details.dataset.persistenceKey = persistenceKey; // For Live Sync

  const isOpen = getToggleState(persistenceKey, defaultOpen);
  details.open = isOpen;
  details.ontoggle = () => setToggleState(persistenceKey, details.open);

  const summary = document.createElement('summary');
  summary.textContent = title;
  details.appendChild(summary);

  const table = document.createElement('table');
  table.className = 'kv-table';
  const tbody = document.createElement('tbody');

  const complexItems = [];

  if (!obj || Object.keys(obj).length === 0) {
    const tr = document.createElement('tr');
    const td = document.createElement('td');
    td.colSpan = 2;
    td.style.color = 'var(--muted)';
    td.style.fontStyle = 'italic';
    td.textContent = 'None';
    tr.appendChild(td);
    tbody.appendChild(tr);
  } else {
    Object.entries(obj).forEach(([k, v]) => {
      // Check for complex value
      if (isComplex(v)) {
        complexItems.push({ key: k, value: v });
      } else {
        const tr = document.createElement('tr');
        const tk = document.createElement('td'); tk.textContent = k;
        const tv = document.createElement('td'); tv.textContent = formatValue(v); tv.className = 'mono';
        tr.appendChild(tk); tr.appendChild(tv); tbody.appendChild(tr);
      }
    });
  }

  // Only append table if it has rows (or "None")
  if (tbody.children.length > 0) {
    table.appendChild(tbody);
    details.appendChild(table);
  }

  wrap.appendChild(details);

  // Append complex items as separate blocks INSIDE the details, but after the table.
  complexItems.forEach(item => {
    const itemSection = createCollapsibleSection(item.key, item.value, `${persistenceKey}:${item.key}`, true);
    itemSection.style.marginTop = '1px';
    details.appendChild(itemSection);
  });

  return wrap;
}

function isComplex(v) {
  if (v === null || v === undefined) return false;
  if (typeof v === 'object') return true; // Array or Object
  if (typeof v === 'string') {
    // Check if it looks like JSON or is long/multiline
    if (v.includes('\n')) return true;
    if ((v.startsWith('{') || v.startsWith('[')) && v.length > 50) return true;
    if (v.length > 200) return true;
  }
  return false;
}

function formatValue(v) {
  if (v === null || v === undefined) return '' + v;
  if (typeof v === 'object') return JSON.stringify(v);
  return String(v);
}

function escapeHtml(str) {
  return String(str).replace(/[&<>]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]));
}

// Floating download/copy/reset/pin actions (non-invasive)
(function setupFloatingActions() {
  const bar = document.createElement('div');
  bar.id = 'floating-actions';
  bar.innerHTML = isPinView ? `
    <button id="fa-download" title="Download NDJSON">⬇</button>
    <button id="fa-copy" title="Copy NDJSON">📋</button>
    <button id="fa-delete" title="Delete pin">🗑</button>
  ` : `
    <button id="fa-download" title="Download NDJSON">⬇</button>
    <button id="fa-copy" title="Copy NDJSON">📋</button>
    <button id="fa-reset" title="Reset session">🔄</button>
    <button id="fa-pin" title="Pin permalink">📌</button>
  `;
  document.body.appendChild(bar);

  document.getElementById('fa-download').onclick = async () => {
    try {
      const text = await fetchStreamText(isPinView ? location.pathname.replace('/p/', '/pin/') : '/stream');
      const blob = new Blob([text], { type: 'application/x-ndjson' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url; a.download = 'sane.ndjson'; document.body.appendChild(a); a.click(); a.remove();
      URL.revokeObjectURL(url);
    } catch (e) { console.error(e); }
  };

  document.getElementById('fa-copy').onclick = async () => {
    try {
      const text = await fetchStreamText(isPinView ? location.pathname.replace('/p/', '/pin/') : '/stream');
      await navigator.clipboard.writeText(text);
      const btn = document.getElementById('fa-copy');
      const prev = btn.textContent;
      btn.textContent = '✓';
      setTimeout(() => btn.textContent = prev, 1000);
    } catch (e) { console.error(e); }
  };

  const resetBtn = document.getElementById('fa-reset');
  if (resetBtn) {
    resetBtn.onclick = async () => {
      try {
        await fetch('/reset', { method: 'POST' });
        records.length = 0;
        renderedIndices.clear();
        openLogIndices.length = 0;
        app.innerHTML = '';
      } catch (e) { console.error(e); }
    };
  }

  const pinBtn = document.getElementById('fa-pin');
  if (pinBtn) {
    pinBtn.onclick = async () => {
      try {
        const text = await fetchStreamText(isPinView ? location.pathname.replace('/p/', '/pin/') : '/stream');
        const res = await fetch('/pin', { method: 'POST', headers: { 'Content-Type': 'application/x-ndjson' }, body: text });
        if (!res.ok) throw new Error('pin failed');
        const { url } = await res.json();
        window.open(url, '_blank');
      } catch (e) { console.error(e); }
    };
  }

  if (isPinView) {
    const delBtn = document.getElementById('fa-delete');
    delBtn.onclick = async () => {
      if (!pinId) return;
      try {
        await fetch(`/pin-delete/${pinId}`, { method: 'POST' });
        alert('Pin deleted');
      } catch (e) { console.error(e); }
    };
  }
})();

let lastKey = '';

// Keybindings: j/k = tab/shift-tab, Shift+j/k = top-level jump, gg/G to first/last, Space toggles, Shift+Space collapse, Shift+D/C/P/R shortcuts
document.addEventListener('keydown', (e) => {
  const inInput = ['INPUT', 'TEXTAREA'].includes(e.target.tagName) || e.target.isContentEditable;
  if (inInput) return;
  const key = e.key.length === 1 ? e.key.toLowerCase() : e.key;
  if (key === 'j' && !e.metaKey && !e.ctrlKey && !e.altKey && !e.shiftKey) { e.preventDefault(); moveFocusLinear(1); lastKey = 'j'; return; }
  if (key === 'k' && !e.metaKey && !e.ctrlKey && !e.altKey && !e.shiftKey) { e.preventDefault(); moveFocusLinear(-1); lastKey = 'k'; return; }
  if (key === 'j' && e.shiftKey && !e.metaKey && !e.ctrlKey && !e.altKey) { e.preventDefault(); moveTopLinear(1); lastKey = 'J'; return; }
  if (key === 'k' && e.shiftKey && !e.metaKey && !e.ctrlKey && !e.altKey) { e.preventDefault(); moveTopLinear(-1); lastKey = 'K'; return; }
  if (key === 'g' && lastKey === 'g' && !e.shiftKey) { e.preventDefault(); focusFirstSummary(); lastKey = ''; return; }
  if (key === 'g' && !e.shiftKey && lastKey !== 'g') { lastKey = 'g'; return; }
  if (key === 'g' && e.shiftKey) { e.preventDefault(); focusLastSummary(); lastKey = ''; return; } // G
  if (e.key === ' ') {
    if (e.shiftKey) { e.preventDefault(); collapseAll(); lastKey = ''; return; }
    lastKey = ' ';
    return;
  }
  if (key === 'd' && e.shiftKey) { e.preventDefault(); document.getElementById('fa-download')?.click(); lastKey = ''; return; }
  if (key === 'c' && e.shiftKey) { e.preventDefault(); document.getElementById('fa-copy')?.click(); lastKey = ''; return; }
  if (key === 'p' && e.shiftKey && !isPinView) { e.preventDefault(); document.getElementById('fa-pin')?.click(); lastKey = ''; return; }
  if (key === 'r' && e.shiftKey && !isPinView) { e.preventDefault(); document.getElementById('fa-reset')?.click(); lastKey = ''; return; }
  lastKey = key;
});

function collectFocusable() {
  // mimic Tab order: natural DOM order of focusable elements that are not hidden
  const selector = 'button, summary, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';
  return Array.from(document.querySelectorAll(selector)).filter(isFocusable);
}

function moveFocusLinear(delta) {
  const focusable = collectFocusable();
  if (!focusable.length) return;
  const idx = focusable.indexOf(document.activeElement);
  let next = idx + delta;
  if (idx === -1) next = delta > 0 ? 0 : focusable.length - 1;
  if (next < 0 || next >= focusable.length) return; // no wrap
  focusable[next].focus();
}

function moveTopLinear(delta) {
  const focusable = Array.from(document.querySelectorAll('details.card > summary')).filter(isFocusableSummary);
  if (!focusable.length) return;
  const idx = focusable.indexOf(document.activeElement);
  let next = idx + delta;
  if (idx === -1) next = delta > 0 ? 0 : focusable.length - 1;
  if (next < 0 || next >= focusable.length) return;
  focusable[next].focus();
}

function focusFirstSummary() {
  const first = document.querySelector('details.card > summary');
  if (first) first.focus();
}
function focusLastSummary() {
  const items = document.querySelectorAll('details.card > summary');
  if (items.length) items[items.length - 1].focus();
}

function collapseAll() {
  document.querySelectorAll('details.card').forEach(d => d.open = false);
  openLogIndices.length = 0;
}

function isFocusable(el) {
  if (el.disabled) return false;
  if (el.getAttribute('aria-hidden') === 'true') return false;
  if (el.tagName === 'SUMMARY') return true;
  return el.offsetParent !== null;
}

function isFocusableSummary(el) {
  if (el.tagName !== 'SUMMARY') return false;
  if (el.getAttribute('aria-hidden') === 'true') return false;
  return true;
}
