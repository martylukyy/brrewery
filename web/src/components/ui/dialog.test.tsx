import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { flushMutations, installReorderingExtension } from "@/test/reordering-extension";

let removeExtension: (() => void) | undefined;

afterEach(() => {
  removeExtension?.();
  removeExtension = undefined;
});

function renderDialog() {
  return render(
    <Dialog open>
      <DialogContent>
        <DialogTitle>Choose version</DialogTitle>
        <DialogDescription>Pick one.</DialogDescription>
      </DialogContent>
    </Dialog>,
  );
}

describe("DialogContent", () => {
  it("keeps the overlay behind the content when <body> children get reordered", async () => {
    removeExtension = installReorderingExtension();

    renderDialog();
    await screen.findByText("Choose version");
    await flushMutations();

    const overlay = document.querySelector('[data-slot="dialog-overlay"]')!;
    const content = document.querySelector('[data-slot="dialog-content"]')!;
    // Both share z-50, so the content only paints above the overlay when it
    // comes later in the document.
    expect(overlay.compareDocumentPosition(content) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("removes its portal container when unmounted", async () => {
    const { unmount } = renderDialog();
    await screen.findByText("Choose version");
    expect(document.querySelector('[data-slot="dialog-portal-root"]')).not.toBeNull();

    unmount();
    expect(document.querySelector('[data-slot="dialog-portal-root"]')).toBeNull();
  });
});
