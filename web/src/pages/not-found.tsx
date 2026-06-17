import { IconArrowLeft } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";

import { Button } from "@/components/ui/button";

export function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background px-4">
      <div className="w-full max-w-md rounded-lg border border-border bg-card p-8 text-center shadow-[0_0_32px_0_var(--color-white)]/5">
        <img src="logos/notfound.png" className="size-32 mx-auto" />
        <h1 className="mt-4 text-xl font-semibold">404 Page not found</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          The page you’re looking for doesn’t exist or may have moved.
        </p>
        <Button asChild className="mt-6">
          <Link to="/">
            <IconArrowLeft className="size-4" />
            Return to dashboard
          </Link>
        </Button>
      </div>
    </div>
  );
}
