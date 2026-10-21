import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@/components/ui/sheet";
import { flushMutations, installReorderingExtension } from "@/test/reordering-extension";

let removeExtension: (() => void) | undefined;

afterEach(() => {
  removeExtension?.();
  removeExtension = undefined;
});

function SidebarSheet({ open }: { open: boolean }) {
  return (
    <Sheet open={open}>
      <SheetContent side="left">
        <SheetTitle>Sidebar</SheetTitle>
        <SheetDescription>Navigation.</SheetDescription>
      </SheetContent>
    </Sheet>
  );
}

describe("SheetContent", () => {
  it("keeps the overlay behind the content when <body> children get reordered", async () => {
    removeExtension = installReorderingExtension();

    render(<SidebarSheet open />);
    await screen.findByText("Sidebar");
    await flushMutations();

    const overlay = document.querySelector('[data-slot="sheet-overlay"]')!;
    const content = document.querySelector('[data-slot="sheet-content"]')!;
    expect(overlay.compareDocumentPosition(content) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  // The mobile sidebar toggles its sheet instead of unmounting it.
  it("mounts overlay and content into its container on every open", async () => {
    const { rerender } = render(<SidebarSheet open={false} />);
    const container = document.querySelector('[data-slot="sheet-portal-root"]')!;
    expect(container.children).toHaveLength(0);

    rerender(<SidebarSheet open />);
    await screen.findByText("Sidebar");
    expect([...container.children].map((child) => child.getAttribute("data-slot")))
      .toEqual(["sheet-overlay", "sheet-content"]);

    rerender(<SidebarSheet open={false} />);
    expect(container.children).toHaveLength(0);

    rerender(<SidebarSheet open />);
    await screen.findByText("Sidebar");
    expect([...container.children].map((child) => child.getAttribute("data-slot")))
      .toEqual(["sheet-overlay", "sheet-content"]);
  });
});
