import { useCallback, useEffect, useRef, useState } from "react";

// Measured widths are rounded down to a multiple of this so that dragging a
// window edge does not produce a new request for every pixel crossed.
const WIDTH_STEP = 32;

// Used until the element has been measured (and in environments without layout,
// such as tests), so charts request a sane number of points on first render.
const FALLBACK_WIDTH = 480;

/**
 * Tracks the rendered width of an element, quantized to keep resizes from
 * churning. Charts use it to ask the daemon for no more points than they have
 * pixels to draw them in.
 */
export function useChartWidth(): {
  ref: (node: HTMLElement | null) => void;
  width: number;
} {
  const [width, setWidth] = useState(FALLBACK_WIDTH);
  const observed = useRef<HTMLElement | null>(null);
  const observer = useRef<ResizeObserver | null>(null);

  useEffect(() => () => observer.current?.disconnect(), []);

  const ref = useCallback((node: HTMLElement | null) => {
    if (node === observed.current) {
      return;
    }
    observed.current = node;
    observer.current?.disconnect();

    if (!node || typeof ResizeObserver === "undefined") {
      return;
    }

    observer.current = new ResizeObserver((entries) => {
      const measured = entries[0]?.contentRect.width ?? 0;
      if (measured > 0) {
        setWidth(Math.max(WIDTH_STEP, Math.floor(measured / WIDTH_STEP) * WIDTH_STEP));
      }
    });
    observer.current.observe(node);
  }, []);

  return { ref, width };
}
