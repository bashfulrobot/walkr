// walkr — client wizard runtime.
// Step content is server-rendered (goldmark, internal/render); this file
// only drives navigation, the shared deep-dive modal, and Mermaid init.
// Adapted from prototype/assets/app.js — see that file's history for the
// dummy-data version this was derived from.

document.addEventListener('alpine:init', () => {
  Alpine.data('walker', () => ({
    current: 0,
    codeOpen: {},
    modal: null,
    steps: window.__REPO_WALKER_STEPS__ || [],

    init() {
      if (window.mermaid) {
        window.mermaid.initialize({
          startOnLoad: false,
          theme: 'dark',
          themeVariables: {
            darkMode: true,
            background: '#101512',
            primaryColor: '#171d19',
            primaryTextColor: '#eae3d1',
            primaryBorderColor: '#e2a33d',
            lineColor: '#6fbdb0',
            secondaryColor: '#232b25',
            tertiaryColor: '#101512',
            fontFamily: "'JetBrains Mono', monospace",
            fontSize: '15px',
          },
          flowchart: { useMaxWidth: false, htmlLabels: true, curve: 'basis' },
          securityLevel: 'strict',
        });
        window.mermaid.run({ querySelector: '.mermaid' });
      }
    },

    go(i) {
      this.current = i;
    },
    next() {
      if (this.current < this.steps.length - 1) this.go(this.current + 1);
    },
    prev() {
      if (this.current > 0) this.go(this.current - 1);
    },
    onKey(e) {
      if (this.modal) return;
      const tag = (e.target.tagName || '').toLowerCase();
      if (tag === 'input' || tag === 'textarea') return;
      if (e.key === 'ArrowRight') this.next();
      if (e.key === 'ArrowLeft') this.prev();
    },
    openModal(id) {
      this.modal = id;
    },
    closeModal() {
      this.modal = null;
    },
  }));
});
