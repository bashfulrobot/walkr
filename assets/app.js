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
      const restored = this.indexForHash();
      if (restored !== -1) this.current = restored;

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
        this.renderMermaid(this.current);
      }
    },

    // Step index matching the current #hash (a step's `id`), or -1 if the
    // hash is empty or doesn't match a known step.
    indexForHash() {
      const id = (location.hash || '').slice(1);
      if (!id) return -1;
      return this.steps.findIndex((s) => s.id === id);
    },

    // Fires on browser Back/Forward and on cross-chapter [text]{step=id}
    // links, which render as plain <a href="#id"> so the browser's native
    // hash navigation lands here before we touch `current`.
    onHashChange() {
      const idx = this.indexForHash();
      if (idx !== -1 && idx !== this.current) {
        this.current = idx;
        this.renderMermaid(idx);
      }
    },

    // Every step's markup is present in the DOM at once (x-show just
    // toggles display:none), but mermaid measures text with getBBox,
    // which returns zero for anything inside a display:none ancestor.
    // Rendering every .mermaid element up front, as mermaid.run()
    // normally does, produces broken/tiny diagrams for every step that
    // isn't current at that instant -- and mermaid marks each element
    // data-processed on its first pass, so it never gets a second
    // chance. So render each step's diagrams lazily, the first time
    // that step is shown.
    //
    // mermaid.run() is also async (it lazy-loads the diagram-type
    // renderer and yields internally), and every call is funneled
    // through mermaid's own single global render queue -- so a render
    // kicked off for step N can still be mid-flight after the user has
    // already clicked past it to step N+1. If step N's <section> has
    // gone back to display:none by then, the same zero-size bug hits.
    // Force the section measurable (but off-flow and unpainted, so
    // there's no visible flash) for exactly the span of its own
    // render, then hand display back to whatever x-show wants it to be
    // by the time the render actually finishes.
    async renderMermaid(stepIndex) {
      if (!window.mermaid) return;
      const section = document.querySelector(`section[data-step-index="${stepIndex}"]`);
      if (!section) return;
      const nodes = Array.from(section.querySelectorAll('.mermaid:not([data-processed])'));
      if (nodes.length === 0) return;
      const forced = section.style.display === 'none';
      if (forced) {
        section.style.display = 'block';
        section.style.position = 'absolute';
        section.style.visibility = 'hidden';
      }
      try {
        await window.mermaid.run({ nodes });
      } finally {
        if (forced) {
          section.style.position = '';
          section.style.visibility = '';
          section.style.display = this.current === stepIndex ? '' : 'none';
        }
      }
    },

    go(i) {
      this.current = i;
      this.renderMermaid(i);
      // Gives every step its own browser-history entry, so Back/Forward --
      // including returning from an external link -- lands on the chapter
      // the reader was actually on, not always chapter one.
      const id = this.steps[i] && this.steps[i].id;
      if (id && location.hash.slice(1) !== id) location.hash = id;
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
