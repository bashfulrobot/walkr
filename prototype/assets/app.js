// walkr — Phase 0 prototype wiring.
// Dummy in-page data only. Real content is authored markdown, rendered server-side by Go.

document.addEventListener('alpine:init', () => {
  Alpine.data('walker', () => ({
    current: 0,
    codeOpen: false,
    modal: null,

    steps: [
      { id: 'overview', title: 'Overview', kind: 'Structure' },
      { id: 'code-walk', title: 'render.go', kind: 'Code walk' },
      { id: 'config', title: 'Deployment', kind: 'Config' },
      { id: 'recap', title: 'Recap', kind: 'Summary' },
    ],

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
      this.codeOpen = false;
    },
    next() {
      if (this.current < this.steps.length - 1) this.go(this.current + 1);
    },
    prev() {
      if (this.current > 0) this.go(this.current - 1);
    },
    onKey(e) {
      if (this.modal) return; // let escape/close handle modal first
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
