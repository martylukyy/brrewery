import { useEffect, useMemo, useState } from "react";

import { ApiError, verifyPassword, type InstallSecret, type PackageStatus } from "@/lib/api";

type Props = {
  packageIds: string[];
  packages: PackageStatus[];
  onClose: () => void;
  onConfirm: (extraVars: Record<string, string>) => void;
};

function requiredSecrets(packages: PackageStatus[], packageIds: string[]): InstallSecret[] {
  const seen = new Set<string>();
  const out: InstallSecret[] = [];

  for (const id of packageIds) {
    const pkg = packages.find((entry) => entry.id === id);
    for (const secret of pkg?.install_secrets ?? []) {
      if (seen.has(secret.key)) {
        continue;
      }
      seen.add(secret.key);
      out.push(secret);
    }
  }

  return out;
}

export function InstallSecretsModal({ packageIds, packages, onClose, onConfirm }: Props) {
  const secrets = useMemo(() => requiredSecrets(packages, packageIds), [packageIds, packages]);
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(secrets.map((secret) => [secret.key, ""])),
  );
  const [error, setError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);

  const packageNames = packageIds
    .map((id) => packages.find((pkg) => pkg.id === id)?.name ?? id)
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

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/60"
        aria-label="Close install credentials dialog"
        onClick={onClose}
      />

      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby="install-secrets-title"
        className="relative z-10 flex w-full max-w-md flex-col rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl"
        onSubmit={handleSubmit}
      >
        <div className="border-b border-zinc-800 px-5 py-4">
          <h2 id="install-secrets-title" className="text-lg font-semibold text-zinc-100">
            Install credentials
          </h2>
          <p className="mt-1 text-sm text-zinc-400">
            Enter the credentials required to install {packageNames}.
          </p>
        </div>

        <div className="space-y-4 px-5 py-4">
          {secrets.map((secret) => (
            <label key={secret.key} className="block">
              <span className="mb-1 block text-sm text-zinc-300">{secret.label}</span>
              <input
                type={secret.type === "password" ? "password" : "text"}
                className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100"
                value={values[secret.key] ?? ""}
                name={secret.key}
                autoComplete={secret.disable_password_manager ? "off" : "current-password"}
                data-1p-ignore={secret.disable_password_manager || undefined}
                data-bwignore={secret.disable_password_manager || undefined}
                data-lpignore={secret.disable_password_manager ? "true" : undefined}
                data-form-type={secret.disable_password_manager ? "other" : undefined}
                onChange={(event) => {
                  setValues((current) => ({
                    ...current,
                    [secret.key]: event.target.value,
                  }));
                }}
              />
            </label>
          ))}
          {error && <p className="text-sm text-red-400">{error}</p>}
        </div>

        <div className="flex justify-end gap-2 border-t border-zinc-800 px-5 py-4">
          <button
            type="button"
            className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800"
            onClick={onClose}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={verifying}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {verifying ? "Verifying…" : "Continue install"}
          </button>
        </div>
      </form>
    </div>
  );
}

export { requiredSecrets };
