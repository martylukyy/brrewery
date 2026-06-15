import { IconLogout, IconServerCog, IconUser } from "@tabler/icons-react";

import { AppIcon } from "@/components/app-icon";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
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
  // Toggle an installed app's systemd service. `enabled` is the requested
  // target state (true = start & enable, false = stop & disable).
  onToggleService: (app: AppStatus, enabled: boolean) => void;
  // The app whose service toggle is awaiting confirmation; its switch is locked
  // until the action resolves so it can't be double-fired.
  pendingServiceAppId?: string;
};

// serviceOn reports whether an app's service switch should read as "on" — the
// service is both running and enabled.
function serviceOn(app: AppStatus): boolean {
  return Boolean(app.service?.active && app.service.enabled);
}

export function AppSidebar({
  apps,
  isLoading = false,
  isError = false,
  errorMessage,
  version,
  user,
  onManageClick,
  onLogout,
  onToggleService,
  pendingServiceAppId,
}: Props) {
  const installed = apps
    .filter((app) => app.installed)
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }));
  const userLabel = user ?? "Signed in";
  const userInitials = user ? initials(user) : null;

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        {/*
          Expanded: logo + name on the left, the collapse toggle pushed to the
          right with ml-auto. The toggle is sized like the app-icon buttons —
          a size-8 (32px) button with a size-6 (24px) glyph — for a consistent
          footprint; [&_svg]:size-6! is needed because the button otherwise pins
          its unclassed icon to size-4. Collapsed: only the toggle remains, and
          being size-8 it fills the rail so its icon lands dead-center; ml-auto
          is then neutralized (no free space) and it settles in place rather than
          jumping the way a justify-center / mx-auto re-center against the
          still-wide rail would. px-0 + transition-[padding] keep that glide.

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
              className="size-6 max-w-none shrink-0 object-contain"
            />
            <span className="font-semibold whitespace-nowrap text-sidebar-foreground">
              brrewery
            </span>
          </div>
          <SidebarTrigger className="ml-auto size-8! [&_svg]:size-6!" />
        </div>
      </SidebarHeader>

      <SidebarSeparator className="mx-4 data-horizontal:w-auto" />

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu className="space-y-1">
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
                  // Reserve room on the right so a long app name doesn't slide
                  // under the absolutely-positioned service switch.
                  const servicePadding = app.service ? "pr-12" : "";

                  return (
                    <SidebarMenuItem key={app.id}>
                      {url ? (
                        // p-1! shrinks the collapsed button's padding so the
                        // size-6 icon fits its 32px slot exactly (centered),
                        // instead of overflowing the default p-2 content box.
                        <SidebarMenuButton
                          asChild
                          tooltip={app.name}
                          className={`group-data-[collapsible=icon]:p-1! ${servicePadding}`}
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
                          className={`group-data-[collapsible=icon]:p-0! ${servicePadding}`}
                        >
                          <AppIcon icon={app.icon} className="size-6 max-w-none" />
                          <span>{app.name}</span>
                        </SidebarMenuButton>
                      )}
                      {app.service && (
                        <ServiceSwitch
                          app={app}
                          pending={pendingServiceAppId === app.id}
                          onToggle={onToggleService}
                        />
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
            <SidebarMenuButton
              tooltip="Manage server"
              onClick={onManageClick}
              className="group-data-[collapsible=icon]:p-1!"
            >
              <IconServerCog className="size-6!" />
              <span>Manage server</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip="Log out"
              onClick={onLogout}
              className="group-data-[collapsible=icon]:p-1!"
            >
              <IconLogout className="size-6!" />
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
    </Sidebar>
  );
}

// ServiceSwitch toggles an installed app's systemd service. It reads "on" when
// the service is running & enabled (thumb right, green) and "off" when stopped
// & disabled (thumb left, red). It sits over the right edge of the menu row and
// is hidden when the sidebar collapses to icons. The toggle is derived from the
// app's service state — flipping it asks the parent to confirm and apply. While
// the transition runs in the background a spinner takes the switch's place, and
// the switch reappears in its new position once the refreshed state arrives.
function ServiceSwitch({
  app,
  pending,
  onToggle,
}: {
  app: AppStatus;
  pending: boolean;
  onToggle: (app: AppStatus, enabled: boolean) => void;
}) {
  const on = serviceOn(app);
  return (
    <div className="absolute top-1/2 right-2 flex -translate-y-1/2 items-center justify-center group-data-[collapsible=icon]:hidden">
      {pending ? (
        <Spinner className="text-sidebar-foreground/70" aria-label={`Updating ${app.name} service`} />
      ) : (
        <Switch
          checked={on}
          onCheckedChange={(checked) => onToggle(app, checked)}
          aria-label={`${on ? "Stop and disable" : "Start and enable"} ${app.name}`}
          // Green when running/enabled, red when stopped/disabled. The dark
          // variants override the base switch's dark-mode unchecked colour.
          className="data-checked:bg-emerald-600 dark:data-checked:bg-emerald-600 data-unchecked:bg-red-600 dark:data-unchecked:bg-red-600"
        />
      )}
    </div>
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
