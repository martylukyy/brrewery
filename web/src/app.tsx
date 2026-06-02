import { AppShell } from "@/components/app-shell";
import { useAuth } from "@/hooks/use-auth";
import { LoginPage } from "@/pages/login";

export function App() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-zinc-400">
        Loading…
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginPage />;
  }

  return <AppShell />;
}
