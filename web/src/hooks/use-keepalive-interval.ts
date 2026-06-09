import { useEffect, useRef } from "react";

// Browsers throttle main-thread setInterval/setTimeout in backgrounded tabs:
// delays are clamped to a ~1s minimum and aligned to second boundaries, then
// dropped to roughly once a minute after a few minutes hidden. Combined with
// React Query rescheduling its poll only after each fetch settles, that turns a
// 1s interval into an effective ~2s (or worse) cadence once the tab loses focus.
//
// Timers inside a dedicated Web Worker are exempt from this throttling, so we
// run the tick there and post a message back on every fire. Where Worker is
// unavailable (jsdom in tests, older runtimes) we fall back to setInterval,
// which still gives the correct cadence whenever the tab is in the foreground.
const WORKER_SOURCE = `
let timer;
self.onmessage = (event) => {
  const data = event.data;
  if (data && data.type === "start") {
    clearInterval(timer);
    timer = setInterval(() => self.postMessage("tick"), data.ms);
  } else if (data && data.type === "stop") {
    clearInterval(timer);
  }
};
`;

/**
 * Invokes `callback` every `ms` milliseconds with a cadence that survives the
 * browser backgrounding the tab. The latest `callback` is always used, so it is
 * safe to pass an inline closure without memoizing it.
 */
export function useKeepaliveInterval(callback: () => void, ms: number): void {
  const savedCallback = useRef(callback);

  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  useEffect(() => {
    const tick = () => savedCallback.current();

    if (typeof Worker === "undefined") {
      const id = setInterval(tick, ms);
      return () => clearInterval(id);
    }

    const url = URL.createObjectURL(
      new Blob([WORKER_SOURCE], { type: "application/javascript" }),
    );
    const worker = new Worker(url);
    worker.onmessage = tick;
    worker.postMessage({ type: "start", ms });

    return () => {
      worker.postMessage({ type: "stop" });
      worker.terminate();
      URL.revokeObjectURL(url);
    };
  }, [ms]);
}
