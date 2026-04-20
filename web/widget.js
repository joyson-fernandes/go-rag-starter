// go-rag-starter widget — vanilla JS, no build step.
//
// Usage on any website:
//   <script src="https://your-bot-host/widget.js"></script>
//
// The script injects a floating bubble + slide-in chat panel. Clicking
// the bubble opens the chat. The panel is non-modal so the host page
// stays fully interactive — users can follow the AI's instructions
// while reading them.
//
// Customise by editing this file and rebuilding the container.

(function () {
  'use strict';

  const script = document.currentScript || document.querySelector('script[src*="widget.js"]');
  const apiBase = (script && script.dataset.api) || window.location.origin;
  const title = (script && script.dataset.title) || 'Ask the Bot';
  const subtitle = (script && script.dataset.subtitle) || 'Answers grounded in the docs';
  const starterPrompts = [
    'How do I replace the corpus?',
    'How do I swap the LLM?',
    'How do I embed this on my site?',
    'What is pgvector?',
  ];

  const css = `
  .ragbot-bubble {
    position: fixed; right: 20px; bottom: 20px; width: 52px; height: 52px;
    border-radius: 50%; background: linear-gradient(135deg, #8b5cf6, #ec4899);
    color: white; border: none; cursor: pointer; z-index: 9999;
    box-shadow: 0 6px 20px rgba(139, 92, 246, 0.4);
    display: flex; align-items: center; justify-content: center;
    transition: transform 0.15s;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  }
  .ragbot-bubble:hover { transform: scale(1.08); }
  .ragbot-bubble svg { width: 22px; height: 22px; }
  .ragbot-panel {
    position: fixed; top: 0; right: 0; width: 100%; max-width: 420px;
    height: 100vh; background: #0b0c14; color: #e7e9f0; z-index: 9998;
    box-shadow: -8px 0 30px rgba(0,0,0,0.3);
    display: flex; flex-direction: column;
    transform: translateX(100%); transition: transform 0.2s ease-out;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    font-size: 14px;
  }
  .ragbot-panel.open { transform: translateX(0); }
  .ragbot-header {
    padding: 16px; border-bottom: 1px solid #222735;
    display: flex; align-items: center; gap: 10px;
  }
  .ragbot-header-icon {
    width: 28px; height: 28px; border-radius: 8px;
    background: linear-gradient(135deg, #8b5cf6, #ec4899);
    display: flex; align-items: center; justify-content: center;
  }
  .ragbot-header-icon svg { width: 16px; height: 16px; color: white; }
  .ragbot-header-text { flex: 1; }
  .ragbot-title { font-weight: 600; font-size: 14px; }
  .ragbot-subtitle { font-size: 11px; color: #8a92a6; margin-top: 2px; }
  .ragbot-close {
    background: transparent; border: none; color: #8a92a6; cursor: pointer;
    padding: 6px; border-radius: 6px; font-size: 18px; line-height: 1;
  }
  .ragbot-close:hover { background: #222735; color: #e7e9f0; }
  .ragbot-messages {
    flex: 1; overflow-y: auto; padding: 16px; display: flex; flex-direction: column; gap: 12px;
  }
  .ragbot-starter { display: flex; flex-direction: column; gap: 6px; margin-top: 8px; }
  .ragbot-starter button {
    text-align: left; background: #141823; color: #e7e9f0;
    border: 1px solid #222735; padding: 8px 12px; border-radius: 8px;
    cursor: pointer; font-size: 12px; font-family: inherit;
  }
  .ragbot-starter button:hover { border-color: #8b5cf6; background: #1a1f2e; }
  .ragbot-msg { display: flex; flex-direction: column; gap: 6px; }
  .ragbot-msg.user { align-items: flex-end; }
  .ragbot-bubble-msg {
    max-width: 90%; padding: 10px 14px; border-radius: 16px;
    white-space: pre-wrap; word-wrap: break-word;
  }
  .ragbot-msg.user .ragbot-bubble-msg {
    background: linear-gradient(135deg, #8b5cf6, #ec4899); color: white;
  }
  .ragbot-msg.assistant .ragbot-bubble-msg { background: #141823; color: #e7e9f0; }
  .ragbot-sources { display: flex; flex-wrap: wrap; gap: 6px; padding: 0 4px; }
  .ragbot-sources-label { font-size: 10px; text-transform: uppercase; letter-spacing: 0.05em; color: #8a92a6; }
  .ragbot-chip {
    background: #141823; border: 1px solid #222735; color: #8a92a6;
    padding: 2px 8px; border-radius: 999px; font-size: 11px;
  }
  .ragbot-input {
    display: flex; gap: 8px; padding: 12px; border-top: 1px solid #222735;
  }
  .ragbot-input textarea {
    flex: 1; min-height: 40px; max-height: 120px; padding: 8px 12px;
    background: #141823; color: #e7e9f0;
    border: 1px solid #222735; border-radius: 8px; font-family: inherit; font-size: 13px;
    resize: none;
  }
  .ragbot-input textarea:focus { outline: none; border-color: #8b5cf6; }
  .ragbot-input button {
    background: linear-gradient(135deg, #8b5cf6, #ec4899); color: white;
    border: none; border-radius: 8px; padding: 0 16px; cursor: pointer;
    font-family: inherit; font-weight: 500;
  }
  .ragbot-input button:disabled { opacity: 0.5; cursor: not-allowed; }
  .ragbot-typing { color: #8a92a6; animation: ragbot-pulse 1.4s infinite; }
  @keyframes ragbot-pulse { 0%,100%{opacity:0.4;} 50%{opacity:1;} }
  `;

  const styleEl = document.createElement('style');
  styleEl.textContent = css;
  document.head.appendChild(styleEl);

  const bubble = document.createElement('button');
  bubble.className = 'ragbot-bubble';
  bubble.setAttribute('aria-label', 'Open chat');
  bubble.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2l2 7h7l-5.5 4 2 7L12 16l-5.5 4 2-7L3 9h7z"/></svg>';

  const panel = document.createElement('aside');
  panel.className = 'ragbot-panel';
  panel.innerHTML = `
    <div class="ragbot-header">
      <div class="ragbot-header-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2l2 7h7l-5.5 4 2 7L12 16l-5.5 4 2-7L3 9h7z"/></svg></div>
      <div class="ragbot-header-text">
        <div class="ragbot-title"></div>
        <div class="ragbot-subtitle"></div>
      </div>
      <button class="ragbot-close" aria-label="Close">×</button>
    </div>
    <div class="ragbot-messages"></div>
    <div class="ragbot-input">
      <textarea rows="1" placeholder="Ask a question..."></textarea>
      <button type="button">Send</button>
    </div>`;

  document.body.appendChild(bubble);
  document.body.appendChild(panel);

  panel.querySelector('.ragbot-title').textContent = title;
  panel.querySelector('.ragbot-subtitle').textContent = subtitle;

  const messagesEl = panel.querySelector('.ragbot-messages');
  const textarea = panel.querySelector('textarea');
  const sendBtn = panel.querySelector('.ragbot-input button');

  let conversationId = null;
  let streaming = false;

  function open() { panel.classList.add('open'); }
  function close() { panel.classList.remove('open'); }
  bubble.addEventListener('click', () => (panel.classList.contains('open') ? close() : open()));
  panel.querySelector('.ragbot-close').addEventListener('click', close);

  renderStarter();

  function renderStarter() {
    messagesEl.innerHTML = '';
    const wrap = document.createElement('div');
    wrap.className = 'ragbot-msg assistant';
    const bubbleMsg = document.createElement('div');
    bubbleMsg.className = 'ragbot-bubble-msg';
    bubbleMsg.textContent = 'Hi! Ask anything — or pick a starter:';
    wrap.appendChild(bubbleMsg);
    const starter = document.createElement('div');
    starter.className = 'ragbot-starter';
    starterPrompts.forEach((p) => {
      const b = document.createElement('button');
      b.textContent = p;
      b.addEventListener('click', () => send(p));
      starter.appendChild(b);
    });
    wrap.appendChild(starter);
    messagesEl.appendChild(wrap);
  }

  const LATEX_MAP = {
    rightarrow: '→', leftarrow: '←', Rightarrow: '⇒', Leftarrow: '⇐',
    to: '→', gets: '←', times: '×', cdot: '·', bullet: '•',
    ldots: '…', dots: '…', checkmark: '✓', pm: '±',
  };
  function clean(s) {
    return s
      .replace(/\$\\([a-zA-Z]+)\$/g, (m, n) => LATEX_MAP[n] || m)
      .replace(/\s*\[\d{2,3}-[a-z0-9-]+\]/g, '')
      .replace(/ +([.,;:])/g, '$1');
  }
  function renderInline(text) {
    const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return esc(text)
      .replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>')
      .replace(/\*([^*\n]+)\*/g, '<em>$1</em>')
      .replace(/`([^`\n]+)`/g, '<code style="background:#222735;padding:1px 4px;border-radius:3px;font-size:12px;">$1</code>');
  }
  function renderMarkdown(text) {
    return text.split('\n').map((line) => {
      const bullet = line.match(/^(\s*)([*-])\s+(.*)$/);
      const numbered = line.match(/^(\s*)(\d+\.)\s+(.*)$/);
      if (bullet) return `${bullet[1]}<span style="color:#8a92a6;">•</span> ${renderInline(bullet[3])}`;
      if (numbered) return `${numbered[1]}<span style="color:#8a92a6;">${numbered[2]}</span> ${renderInline(numbered[3])}`;
      return renderInline(line);
    }).join('\n');
  }
  function formatSource(path) {
    return path.replace(/^docs\//, '').replace(/\.md$/, '').replace(/^\d+-/, '');
  }

  sendBtn.addEventListener('click', () => send(textarea.value));
  textarea.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(textarea.value); }
  });

  function send(text) {
    const q = text.trim();
    if (!q || streaming) return;
    textarea.value = '';
    streaming = true;
    sendBtn.disabled = true;

    if (messagesEl.querySelector('.ragbot-starter')) messagesEl.innerHTML = '';

    appendMessage('user', q);
    const asst = appendMessage('assistant', '');
    const typing = document.createElement('span');
    typing.className = 'ragbot-typing';
    typing.textContent = '…';
    asst.bubble.appendChild(typing);

    let full = '';
    let sources = [];

    fetch(`${apiBase}/api/query`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: q, conversation_id: conversationId || undefined }),
    }).then(async (res) => {
      if (!res.ok || !res.body) {
        asst.bubble.textContent = `Error: HTTP ${res.status}`;
        streaming = false; sendBtn.disabled = false;
        return;
      }
      const reader = res.body.getReader();
      const dec = new TextDecoder();
      let buf = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true });
        let idx;
        while ((idx = buf.indexOf('\n\n')) !== -1) {
          const raw = buf.slice(0, idx);
          buf = buf.slice(idx + 2);
          let ev = 'message', data = '';
          raw.split('\n').forEach((l) => {
            if (l.startsWith('event:')) ev = l.slice(6).trim();
            else if (l.startsWith('data:')) data += l.slice(5).trim();
          });
          if (!data) continue;
          let p;
          try { p = JSON.parse(data); } catch { continue; }
          if (ev === 'meta') {
            conversationId = p.conversation_id;
            sources = p.sources || [];
          } else if (ev === 'token') {
            if (typing.parentNode) typing.remove();
            full += p.t;
            asst.bubble.innerHTML = renderMarkdown(clean(full));
            messagesEl.scrollTop = messagesEl.scrollHeight;
          } else if (ev === 'done') {
            sources = p.sources || sources;
            renderSources(asst.wrap, sources);
            streaming = false; sendBtn.disabled = false;
          } else if (ev === 'error') {
            asst.bubble.textContent = `Error: ${p.error || 'unknown'}`;
            streaming = false; sendBtn.disabled = false;
          }
        }
      }
      streaming = false; sendBtn.disabled = false;
    }).catch((err) => {
      asst.bubble.textContent = `Error: ${err.message || 'network error'}`;
      streaming = false; sendBtn.disabled = false;
    });
  }

  function appendMessage(role, content) {
    const wrap = document.createElement('div');
    wrap.className = `ragbot-msg ${role}`;
    const bubble = document.createElement('div');
    bubble.className = 'ragbot-bubble-msg';
    bubble.textContent = content;
    wrap.appendChild(bubble);
    messagesEl.appendChild(wrap);
    messagesEl.scrollTop = messagesEl.scrollHeight;
    return { wrap, bubble };
  }

  function renderSources(wrap, sources) {
    if (!sources || !sources.length) return;
    const row = document.createElement('div');
    row.className = 'ragbot-sources';
    const label = document.createElement('span');
    label.className = 'ragbot-sources-label';
    label.textContent = 'Sources';
    row.appendChild(label);
    sources.forEach((s) => {
      const chip = document.createElement('span');
      chip.className = 'ragbot-chip';
      chip.textContent = formatSource(s);
      row.appendChild(chip);
    });
    wrap.appendChild(row);
  }
})();
