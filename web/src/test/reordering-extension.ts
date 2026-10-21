// Mimics a browser extension (seen with a password manager) that keeps its own
// injected element last in <body>: every node added after it is moved back in
// front of it, last-first, which reverses the order of nodes added together.
// Returns a cleanup function.
export function installReorderingExtension(): () => void {
  const own = document.createElement("ext-sentinel");
  document.body.appendChild(own);
  const observer = new MutationObserver((records) => {
    const added = records
      .flatMap((record) => [...record.addedNodes])
      .filter((node) => node.parentNode === document.body &&
        own.compareDocumentPosition(node) & Node.DOCUMENT_POSITION_FOLLOWING);
    for (const node of added.reverse()) {
      document.body.insertBefore(node, own);
    }
  });
  observer.observe(document.body, { childList: true });
  return () => {
    observer.disconnect();
    own.remove();
  };
}

// Lets pending MutationObserver callbacks run.
export function flushMutations(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}
