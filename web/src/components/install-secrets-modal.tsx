import { useMemo, useState } from "react";

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
import { ApiError, verifyPassword, type AppStatus, type AppJobAction } from "@/lib/api";
import { requiredSecrets } from "@/lib/install-secrets";

type Props = {
  action: AppJobAction;
  appIds: string[];
  apps: AppStatus[];
  onClose: () => void;
  onConfirm: (extraVars: Record<string, string>) => void;
};

const ACTION_COPY: Record<AppJobAction, { title: string; verb: string; submit: string }> = {
  install: { title: "Install credentials", verb: "install", submit: "Continue install" },
  upgrade: { title: "Confirm your password", verb: "upgrade", submit: "Continue upgrade" },
  remove: { title: "Confirm your password", verb: "remove", submit: "Continue remove" },
};

export function InstallSecretsModal({ action, appIds, apps, onClose, onConfirm }: Props) {
  const secrets = useMemo(() => requiredSecrets(apps, appIds, action), [action, appIds, apps]);
  const copy = ACTION_COPY[action];
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(secrets.map((secret) => [secret.key, ""])),
  );
  const [error, setError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);

  const appNames = appIds
    .map((id) => apps.find((app) => app.id === id)?.name ?? id)
    .join(", ");

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);

    for (const secret of secrets) {
      if (!values[secret.key]?.trim()) {
        setError(`${secret.label} is required.`);
        return;
      }
    }

    // Confirm any account password matches the real Linux / brrewery account
    // before proceeding, so a typo is caught here instead of failing the install.
    setVerifying(true);
    try {
      for (const secret of secrets) {
        if (secret.verify_brrewery_password) {
          await verifyPassword(values[secret.key] ?? "");
        }
      }
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 401
          ? "Incorrect password."
          : "Could not verify the password. Please try again.",
      );
      return;
    } finally {
      setVerifying(false);
    }

    onConfirm(
      Object.fromEntries(secrets.map((secret) => [secret.key, values[secret.key] ?? ""])),
    );
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[90vh] flex-col gap-0 p-0 sm:max-w-md">
        <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
          <DialogHeader className="gap-1 border-b border-border px-5 py-4">
            <DialogTitle className="text-base">{copy.title}</DialogTitle>
            <DialogDescription>
              Enter the credentials required to {copy.verb} {appNames}.
            </DialogDescription>
          </DialogHeader>

          <div className="scrollbar-zinc min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
            {secrets.map((secret) => {
              // Suppress browser/extension password managers for any password
              // field — autofilling or saving an app's install credential is
              // never wanted — as well as when explicitly requested per secret.
              const disablePwManager =
                secret.disable_password_manager || secret.type === "password";
              return (
                <div key={secret.key} className="space-y-1">
                  <Label htmlFor={`secret-${secret.key}`}>{secret.label}</Label>
                  <Input
                    id={`secret-${secret.key}`}
                    type={secret.type === "password" ? "password" : "text"}
                    value={values[secret.key] ?? ""}
                    name={secret.key}
                    autoComplete={disablePwManager ? "off" : "current-password"}
                    data-1p-ignore={disablePwManager || undefined}
                    data-bwignore={disablePwManager || undefined}
                    data-lpignore={disablePwManager ? "true" : undefined}
                    data-form-type={disablePwManager ? "other" : undefined}
                    onChange={(event) => {
                      setValues((current) => ({
                        ...current,
                        [secret.key]: event.target.value,
                      }));
                    }}
                  />
                </div>
              );
            })}
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>

          <DialogFooter className="border-t border-border px-5 py-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={verifying}>
              {verifying ? "Verifying…" : copy.submit}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
