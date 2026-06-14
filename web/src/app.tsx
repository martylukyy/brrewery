import { AppShell } from "@/components/app-shell";
import { Spinner } from "@/components/ui/spinner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useAuth } from "@/hooks/use-auth";
import { LoginPage } from "@/pages/login";

export function App() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center gap-2 text-muted-foreground">
        <Spinner />
        Loading…
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginPage />;
  }

  return (
    <TooltipProvider>
      <AppShell />
    </TooltipProvider>
  );
}
