import "@testing-library/jest-dom/vitest";

// jsdom does not implement a handful of browser APIs that Radix UI primitives
// (used by the shadcn components) rely on. Provide no-op shims so dialogs,
// selects, tooltips, etc. can render in the test environment.
if (typeof window !== "undefined" && !window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }) as unknown as MediaQueryList;
}

if (typeof window !== "undefined" && !window.ResizeObserver) {
  // recharts' ResponsiveContainer sizes itself from the ResizeObserver entry's
  // contentRect; jsdom has no layout, so report a fixed non-zero size on observe
  // to let charts render in tests.
  window.ResizeObserver = class {
    constructor(private callback: ResizeObserverCallback) {}
    observe(target: Element) {
      const contentRect = { width: 640, height: 320, top: 0, left: 0, right: 640, bottom: 320, x: 0, y: 0 } as DOMRectReadOnly;
      this.callback([{ target, contentRect } as ResizeObserverEntry], this);
    }
    unobserve() {}
    disconnect() {}
  };
}

if (typeof Element !== "undefined") {
  Element.prototype.scrollIntoView ??= () => {};
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
}
