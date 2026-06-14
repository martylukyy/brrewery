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
        <div className="flex items-center gap-2 px-1 py-1">
          <img
            src="/brrewery.png"
            alt=""
            className="size-8 shrink-0 rounded-lg object-contain"
          />
          <span className="font-semibold text-sidebar-foreground group-data-[collapsible=icon]:hidden">
            brrewery
          </span>
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
                        <SidebarMenuButton asChild tooltip={app.name}>
                          <a href={url} target="_blank" rel="noopener noreferrer">
                            <AppIcon icon={app.icon} className="size-6" />
                            <span>{app.name}</span>
                          </a>
                        </SidebarMenuButton>
                      ) : (
                        // Installed but no web UI to link to — show it, but inert.
                        <SidebarMenuButton disabled tooltip={app.name}>
                          <AppIcon icon={app.icon} className="size-6" />
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
