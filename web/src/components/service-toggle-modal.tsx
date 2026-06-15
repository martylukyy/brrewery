import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { AppStatus } from "@/lib/api";

type Props = {
  app: AppStatus;
  // The target state: true starts & enables the service, false stops & disables it.
  enabled: boolean;
  onClose: () => void;
  // Hand the entered password back to the parent, which closes this modal and
  // runs the transition in the background (a spinner replaces the switch).
  onConfirm: (password: string) => void;
};

export function ServiceToggleModal({ app, enabled, onClose, onConfirm }: Props) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const verb = enabled ? "start and enable" : "stop and disable";
  const submitLabel = enabled ? "Start service" : "Stop service";

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    if (!password.trim()) {
      setError("Account password is required.");
      return;
    }
    onConfirm(password);
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[90vh] flex-col gap-0 p-0 sm:max-w-md">
        <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
          <DialogHeader className="gap-1 border-b border-border px-5 py-4">
            <DialogTitle className="text-base">Confirm your password</DialogTitle>
            <DialogDescription>
              Enter your account password to {verb} {app.name}.
            </DialogDescription>
          </DialogHeader>

          <div className="scrollbar-zinc min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
            <div className="space-y-1">
              <Label htmlFor="service-password">Password</Label>
              <Input
                id="service-password"
                type="password"
                value={password}
                name="service-password"
                autoComplete="current-password"
                autoFocus
                onChange={(event) => {
                  setPassword(event.target.value);
                  setError(null);
                }}
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>

          <DialogFooter className="border-t border-border px-5 py-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" variant={enabled ? "default" : "destructive"}>
              {submitLabel}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
