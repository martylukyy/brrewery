import { useState } from "react";
import { IconDownload, IconX } from "@tabler/icons-react";

import { Button } from "@/components/ui/button";
import { SidebarMenuButton } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";

type Props = {
  // The available release version (e.g. "1.2.0").
  latestVersion?: string;
  // Opens the in-app update modal (brrewery installs updates itself, so the
  // action is "Install Update" rather than a link to the release page).
  onInstall?: () => void;
};

// UpdateBanner is the sidebar's update notification, styled after qui's
// UpdateBanner: a dismissible green card with the version and an action
// button. Dismissal is session-local; the banner returns on the next page
// load while the release is still newer.
//
// The card needs the expanded sidebar's width, so when the sidebar collapses
// to the icon rail it is swapped for a compact icon-only button in the same
// slot, keeping the notice visible in both modes.
export function UpdateBanner({ latestVersion, onInstall }: Props) {
  const [dismissed, setDismissed] = useState(false);

  if (dismissed) {
    return null;
  }

  return (
    <>
      <div
        className={cn(
          "rounded-md border border-green-200 bg-green-50 m-2 p-2",
          "dark:border-green-800 dark:bg-green-950/50",
          "group-data-[collapsible=icon]:hidden",
        )}
      >
        <div className="flex items-start gap-2">
          <IconDownload className="mt-0.5 size-4 shrink-0 text-green-600 dark:text-green-400" />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-green-800 dark:text-green-200">
              Update Available
            </p>
            <p className="mt-1 text-xs text-green-700 dark:text-green-300">
              {latestVersion ? `Version ${latestVersion}` : "A new version"}
            </p>
            <p className="mt-1 text-xs text-green-700 dark:text-green-300">
              is available!
            </p>
            <Button
              size="sm"
              variant="outline"
              className="mt-2 h-6 border-green-300 text-xs text-green-700 hover:bg-green-100 dark:border-green-700 dark:text-green-300 dark:hover:bg-green-900"
              onClick={onInstall}
            >
              Install Update
            </Button>
          </div>
          <Button
            size="icon"
            variant="ghost"
            className="h-4 w-4 text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-200"
            onClick={() => setDismissed(true)}
          >
            <IconX className="h-3 w-3" />
            <span className="sr-only">Dismiss</span>
          </Button>
        </div>
      </div>

      {/* Icon-rail fallback: the banner card is hidden when collapsed, so a
          compact green button carries the notice there instead. */}
      <SidebarMenuButton
        tooltip={`Update brrewery to ${latestVersion ?? "the latest version"}`}
        onClick={onInstall}
        className="hidden text-green-600 hover:text-green-700 group-data-[collapsible=icon]:flex group-data-[collapsible=icon]:p-1! dark:text-green-400 dark:hover:text-green-300"
      >
        <IconDownload className="size-6!" />
        <span>Update Available</span>
      </SidebarMenuButton>
    </>
  );
}
