import { useEffect } from "react";
import { Outlet, useNavigate } from "@tanstack/react-router";

import { AppShell } from "@/components/app-shell";
import { Spinner } from "@/components/ui/spinner";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useAuth } from "@/hooks/use-auth";
import { LoginPage } from "@/pages/login";

function FullScreenSpinner() {
  return (
    <div className="flex min-h-screen items-center justify-center gap-2 text-muted-foreground">
      <Spinner />
      Loading…
    </div>
  );
}

export function RootLayout() {
  return (
    <TooltipProvider>
      <Outlet />
      <Toaster />
    </TooltipProvider>
  );
}

// RequireAuth gates the dashboard: while the session probe is in flight it shows
// a spinner, and an unauthenticated session is bounced to /login. It reacts to
// useAuth so a background 401 that clears the session cache mid-use redirects
// too (not just direct/initial navigation) — preserving the prior behaviour
// where a stale cookie routed back to login instead of leaving a dead page.
function RequireAuth({ children }: { children: React.ReactNode }) {
  const { isLoading, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      void navigate({ to: "/login" });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading || !isAuthenticated) {
    return <FullScreenSpinner />;
  }
  return <>{children}</>;
}

export function DashboardRoute() {
  return (
    <RequireAuth>
      <AppShell />
    </RequireAuth>
  );
}

// LoginRoute is the inverse guard: an already-authenticated visitor — including
// the moment after signing in, which populates the session cache — is sent on
// to the dashboard.
export function LoginRoute() {
  const { isLoading, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      void navigate({ to: "/" });
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading || isAuthenticated) {
    return <FullScreenSpinner />;
  }
  return <LoginPage />;
}
