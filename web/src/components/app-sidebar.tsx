import { IconLogout, IconServerCog, IconUser } from "@tabler/icons-react";

import { AppIcon } from "@/components/app-icon";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
  SidebarRail,
  SidebarSeparator,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { appUrl } from "@/lib/app-link";
import type { AppStatus } from "@/lib/api";

type Props = {
  apps: AppStatus[];
  isLoading?: boolean;
  isError?: boolean;
  errorMessage?: string;
  version?: string;
  // The signed-in user's name/email. The current session model (a cookie-backed
  // VersionInfo) exposes no identity field, so this is usually undefined and the
  // footer falls back to a generic "Signed in" label rather than inventing one.
  user?: string;
  onManageClick: () => void;
  onLogout: () => void;
};

export function AppSidebar({
  apps,
  isLoading = false,
  isError = false,
  errorMessage,
  version,
  user,
  onManageClick,
  onLogout,
}: Props) {
  const installed = apps.filter((app) => app.installed);
  const userLabel = user ?? "Signed in";
  const userInitials = user ? initials(user) : null;

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        {/*
          Expanded: logo + name on the left, the collapse toggle pushed to the
          right with ml-auto. Collapsed: only the toggle remains; it grows to
          size-8 like the app-icon buttons so it fills the rail and its icon
          lands dead-center. (ml-auto alone right-aligns the 28px button, leaving
          a lopsided left margin.) px-0 + transition-[padding] let it glide into
          place as the rail width animates, instead of jumping the way a
          justify-center / mx-auto re-center against the still-wide rail would.

          The logo + name sit in their own overflow-hidden box that collapses its
          max-width to 0 instead of using display:none. data-collapsible flips at
          the very start of the width animation, so a display toggle would snap
          them to full size in the still-narrow rail and bleed into the content;
          animating max-width + opacity instead wipes/fades them in step with the
          rail. Collapsing to 0 width also keeps them from shoving the toggle.
        */}
        <div className="flex items-center px-2 py-1 transition-[padding] group-data-[collapsible=icon]:px-0">
          <div className="flex max-w-48 items-center gap-2 overflow-hidden transition-[max-width,opacity] duration-200 ease-linear group-data-[collapsible=icon]:max-w-0 group-data-[collapsible=icon]:opacity-0">
            <img
              src="/logos/brrewery.webp"
              alt=""
              // max-w-none defeats Preflight's `img { max-width: 100% }`, which
              // would otherwise clamp the logo's width to its narrow parent.
              className="size-8 max-w-none shrink-0 object-contain"
            />
            <span className="font-semibold whitespace-nowrap text-sidebar-foreground">
              brrewery
            </span>
          </div>
          <SidebarTrigger className="ml-auto group-data-[collapsible=icon]:size-8!" />
        </div>
      </SidebarHeader>

      <SidebarSeparator className="mx-4 data-horizontal:w-auto" />

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {isLoading ? (
                [0, 1, 2, 3].map((i) => (
                  <SidebarMenuItem key={i}>
                    <SidebarMenuSkeleton showIcon />
                  </SidebarMenuItem>
                ))
              ) : isError ? (
                <p className="px-2 py-1.5 text-xs text-destructive group-data-[collapsible=icon]:hidden">
                  {errorMessage ?? "Failed to load apps."}
                </p>
              ) : installed.length === 0 ? (
                <p className="px-2 py-1.5 text-xs text-sidebar-foreground/70 group-data-[collapsible=icon]:hidden">
                  No apps installed
                </p>
              ) : (
                installed.map((app) => {
                  const url = appUrl(app.web_path);

                  return (
                    <SidebarMenuItem key={app.id}>
                      {url ? (
                        // p-1! shrinks the collapsed button's padding so the
                        // size-6 icon fits its 32px slot exactly (centered),
                        // instead of overflowing the default p-2 content box.
                        <SidebarMenuButton
                          asChild
                          tooltip={app.name}
                          className="group-data-[collapsible=icon]:p-1!"
                        >
                          <a href={url} target="_blank" rel="noopener noreferrer">
                            <AppIcon icon={app.icon} className="size-6 max-w-none" />
                            <span>{app.name}</span>
                          </a>
                        </SidebarMenuButton>
                      ) : (
                        // Installed but no web UI to link to — show it, but inert.
                        <SidebarMenuButton
                          disabled
                          tooltip={app.name}
                          className="group-data-[collapsible=icon]:p-1!"
                        >
                          <AppIcon icon={app.icon} className="size-6 max-w-none" />
                          <span>{app.name}</span>
                        </SidebarMenuButton>
                      )}
                    </SidebarMenuItem>
                  );
                })
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarSeparator className="mx-4 data-horizontal:w-auto" />

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Manage server" onClick={onManageClick}>
              <IconServerCog />
              <span>Manage server</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Log out" onClick={onLogout}>
              <IconLogout />
              <span>Log out</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>

      <SidebarSeparator className="mx-4 data-horizontal:w-auto" />

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" tooltip={userLabel} className="cursor-default">
              <Avatar className="size-8 rounded-lg after:rounded-lg">
                <AvatarFallback className="rounded-lg">
                  {userInitials ?? <IconUser className="size-4" />}
                </AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
                <span className="truncate font-medium">{userLabel}</span>
                {version && (
                  <span className="truncate text-xs text-sidebar-foreground/70">
                    Version {version}
                  </span>
                )}
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  );
}

// initials derives one uppercase letter from a name or email for use as an
// avatar fallback (e.g. "stefan.luksch" -> "S", "Stefan Luksch" -> "S").
function initials(value: string): string {
  const local = value.includes("@") ? value.split("@")[0] : value;
  const parts = local.split(/[\s._-]+/).filter(Boolean);
  const letter = parts.length > 0 ? parts[0][0] : value[0];
  return letter.toUpperCase();
}
