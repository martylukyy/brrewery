import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  ApiError,
  applySysctl,
  getSysctl,
  verifyPassword,
  type SysctlReport,
  type SysctlSetting,
} from "@/lib/api";

type Props = {
  onClose: () => void;
};

type Group = {
  name: string;
  settings: SysctlSetting[];
};

// groupSettings buckets settings by their group, preserving the order in which
// each group first appears. The backend returns the catalog sorted by group then
// key, so groups render alphabetically with keys ordered within each.
function groupSettings(settings: SysctlSetting[]): Group[] {
  const groups: Group[] = [];
  const byName = new Map<string, Group>();
  for (const setting of settings) {
    let group = byName.get(setting.group);
    if (!group) {
      group = { name: setting.group, settings: [] };
      byName.set(setting.group, group);
      groups.push(group);
    }
    group.settings.push(setting);
  }
  return groups;
}

function seedValue(setting: SysctlSetting): string {
  // Unavailable keys (absent on this kernel) have no live value, so show an empty
  // field rather than the recommended value the user has not actually chosen.
  return setting.available ? setting.value : "";
}

// parseSysctlPatch reads a sysctl.conf-style file ("key = value" per line, with
// "#"/";" comments) into a key→value map. Values keep their internal spaces so
// multi-field settings (e.g. tcp_rmem) survive; the server validates them.
function parseSysctlPatch(text: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const rawLine of text.split("\n")) {
    const line = rawLine.replace(/\r$/, "").trim();
    if (!line || line.startsWith("#") || line.startsWith(";")) {
      continue;
    }
    const eq = line.indexOf("=");
    if (eq < 0) {
      continue;
    }
    const key = line.slice(0, eq).trim();
    const value = line.slice(eq + 1).trim();
    if (key && value) {
      out[key] = value;
    }
  }
  return out;
}

export function SysctlModal({ onClose }: Props) {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["sysctl"], queryFn: getSysctl });
  const report = query.data;

  // `edits` holds only the fields the user has changed; the value shown for any
  // untouched field is derived from the live report (see effectiveValue). This
  // avoids seeding state from an effect and keeps inputs in sync after a refetch.
  const [edits, setEdits] = useState<Record<string, string>>({});
  const [applied, setApplied] = useState(false);
  // The password is collected by a prompt shown on Apply / Upload patch (mirroring
  // the app install flow) rather than entered inline in the footer.
  const [promptOpen, setPromptOpen] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const effectiveValue = (setting: SysctlSetting): string =>
    edits[setting.key] ?? seedValue(setting);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  const groups = useMemo(() => groupSettings(report?.settings ?? []), [report]);
  const writable = report?.writable ?? false;

  const apply = useMutation({
    mutationFn: (password: string) => {
      // Persist every readable parameter so the managed drop-in always reflects
      // the full panel; unreadable keys (absent on this kernel) are left alone.
      const payload: Record<string, string> = {};
      for (const setting of report?.settings ?? []) {
        if (setting.available) {
          payload[setting.key] = effectiveValue(setting);
        }
      }
      return applySysctl({ values: payload, password });
    },
    onSuccess: (fresh: SysctlReport) => {
      // Drop local edits so inputs re-derive from the freshly applied values.
      queryClient.setQueryData(["sysctl"], fresh);
      setEdits({});
      setApplied(true);
    },
  });

  function update(key: string, value: string) {
    setApplied(false);
    setEdits((current) => ({ ...current, [key]: value }));
  }

  // Both Apply and Upload patch converge on the password prompt; on confirm the
  // current panel values (Upload patch having merged the file in first) are applied.
  function requestApply() {
    setUploadError(null);
    setPromptOpen(true);
  }

  async function handlePatchFile(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    // Reset so selecting the same file again still fires onChange.
    event.target.value = "";
    if (!file) {
      return;
    }

    const parsed = parseSysctlPatch(await file.text());
    const known = new Set((report?.settings ?? []).map((setting) => setting.key));
    const matched: Record<string, string> = {};
    for (const [key, value] of Object.entries(parsed)) {
      if (known.has(key)) {
        matched[key] = value;
      }
    }

    if (Object.keys(matched).length === 0) {
      setApplied(false);
      setUploadError("No recognized sysctl parameters were found in that file.");
      return;
    }

    setApplied(false);
    setUploadError(null);
    setEdits((current) => ({ ...current, ...matched }));
    setPromptOpen(true);
  }

  function confirmPassword(password: string) {
    setPromptOpen(false);
    apply.mutate(password);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60" aria-hidden="true" />

      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="sysctl-title"
        className="relative z-10 flex h-full sm:max-h-[90%] w-full sm:max-w-[60%] flex-col rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl"
      >
        <div className="flex items-start justify-between gap-4 border-b border-zinc-800 px-5 py-4">
          <div>
            <h2 id="sysctl-title" className="text-lg font-semibold text-zinc-100">
              Tune sysctl parameters
            </h2>
            <p className="mt-1 text-sm text-zinc-400">
              Adjust kernel parameters for network and storage throughput on this host.
            </p>
          </div>
          <button
            type="button"
            className="-mr-1 -mt-1 shrink-0 rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-zinc-100"
            aria-label="Close tune sysctl dialog"
            onClick={onClose}
          >
            <svg
              viewBox="0 0 24 24"
              className="size-6"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              aria-hidden="true"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </div>

        <div className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto px-5 py-3">
          {query.isLoading && <p className="py-6 text-center text-sm text-zinc-500">Loading parameters…</p>}
          {query.isError && (
            <p className="py-6 text-center text-sm text-red-400">{(query.error as Error).message}</p>
          )}

          {report && !writable && (
            <p className="mb-3 rounded-md border border-amber-900/60 bg-amber-950/30 px-3 py-2 text-xs text-amber-300">
              Applying changes is not supported on this platform. Values are shown for reference only.
            </p>
          )}

          {groups.map((group) => (
            <section key={group.name} className="mb-5">
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                {group.name}
              </h3>
              <ul className="space-y-1">
                {group.settings.map((setting) => (
                  <SettingRow
                    key={setting.key}
                    setting={setting}
                    value={effectiveValue(setting)}
                    disabled={!writable || !setting.available}
                    onChange={(value) => update(setting.key, value)}
                    onReset={() => update(setting.key, setting.recommended)}
                  />
                ))}
              </ul>
            </section>
          ))}
        </div>

        <div className="border-t border-zinc-800 px-5 py-4">
          <input
            ref={fileInputRef}
            type="file"
            accept=".conf,.txt,text/plain"
            className="hidden"
            aria-hidden="true"
            tabIndex={-1}
            onChange={handlePatchFile}
          />

          <div className="flex items-center justify-between gap-3">
            <p className="min-w-0 flex-1 truncate text-sm">
              {uploadError && <span className="text-red-400">{uploadError}</span>}
              {apply.isError && (
                <span className="text-red-400">{(apply.error as Error).message}</span>
              )}
              {applied && <span className="text-emerald-400">Settings applied.</span>}
            </p>
            <div className="flex shrink-0 items-center gap-3">
              <button
                type="button"
                className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={!writable || apply.isPending}
                onClick={() => fileInputRef.current?.click()}
              >
                Upload patch
              </button>
              <button
                type="button"
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={!writable || apply.isPending}
                onClick={requestApply}
              >
                {apply.isPending ? "Applying…" : "Apply"}
              </button>
            </div>
          </div>
        </div>
      </div>

      {promptOpen && (
        <SysctlPasswordPrompt onCancel={() => setPromptOpen(false)} onConfirm={confirmPassword} />
      )}
    </div>
  );
}

type PasswordPromptProps = {
  onCancel: () => void;
  onConfirm: (password: string) => void;
};

// SysctlPasswordPrompt mirrors the install-credentials dialog: it collects the
// operator's account password, verifies it up front (catching typos before the
// privileged apply), then hands it back to the caller.
function SysctlPasswordPrompt({ onCancel, onConfirm }: PasswordPromptProps) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);

    if (!password.trim()) {
      setError("Password is required.");
      return;
    }

    setVerifying(true);
    try {
      await verifyPassword(password);
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

    onConfirm(password);
  }

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onCancel();
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onCancel]);

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/60"
        aria-label="Close password dialog"
        onClick={onCancel}
      />

      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby="sysctl-password-title"
        className="relative z-10 flex w-full max-w-md flex-col rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl"
        onSubmit={handleSubmit}
      >
        <div className="border-b border-zinc-800 px-5 py-4">
          <h2 id="sysctl-password-title" className="text-lg font-semibold text-zinc-100">
            Confirm your password
          </h2>
          <p className="mt-1 text-sm text-zinc-400">
            Enter your account password to apply the sysctl changes.
          </p>
        </div>

        <div className="space-y-4 px-5 py-4">
          <label className="block">
            <span className="mb-1 block text-sm text-zinc-300">Account password</span>
            <input
              type="password"
              className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100"
              value={password}
              name="password"
              autoComplete="current-password"
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>
          {error && <p className="text-sm text-red-400">{error}</p>}
        </div>

        <div className="flex justify-end gap-2 border-t border-zinc-800 px-5 py-4">
          <button
            type="button"
            className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800"
            onClick={onCancel}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={verifying}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {verifying ? "Verifying…" : "Continue"}
          </button>
        </div>
      </form>
    </div>
  );
}

type RowProps = {
  setting: SysctlSetting;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
  onReset: () => void;
};

function SettingRow({ setting, value, disabled, onChange, onReset }: RowProps) {
  const current = setting.available ? setting.value : "unavailable";

  return (
    <li className="flex flex-col gap-1 rounded-md px-2 py-2 hover:bg-zinc-800/40 sm:flex-row sm:items-start sm:gap-4">
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <span className="text-sm font-medium text-zinc-100">{setting.label}</span>
          {setting.unit && <span className="text-[10px] text-zinc-500">in {setting.unit}</span>}
        </div>
        <p className="mt-0.5 text-xs text-zinc-500">{setting.description}</p>
        <p className="mt-0.5 font-mono text-[10px] text-zinc-600">{setting.key}</p>
      </div>

      <div className="flex shrink-0 flex-col items-stretch gap-1 sm:w-80
      ">
        {setting.kind === "enum" && setting.choices ? (
          <select
            className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1.5 text-sm text-zinc-100 disabled:opacity-50"
            aria-label={setting.label}
            value={value}
            disabled={disabled}
            onChange={(event) => onChange(event.target.value)}
          >
            {/* Surface the live value even when it is outside the recommended choices. */}
            {setting.choices.includes(value) ? null : <option value={value}>{value || "—"}</option>}
            {setting.choices.map((choice) => (
              <option key={choice} value={choice}>
                {choice}
              </option>
            ))}
          </select>
        ) : (
          <input
            type="text"
            inputMode={setting.kind === "integer" ? "numeric" : "text"}
            className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1.5 text-sm text-zinc-100 disabled:opacity-50"
            aria-label={setting.label}
            placeholder={setting.available ? undefined : "unavailable"}
            value={value}
            disabled={disabled}
            onChange={(event) => onChange(event.target.value)}
          />
        )}
        <div className="flex items-center justify-between text-[10px] text-zinc-500">
          <span className="truncate" title={`Current: ${current}`}>
            now: {current || "—"}
          </span>
          <button
            type="button"
            className="rounded px-1 text-blue-400 hover:text-blue-300 disabled:opacity-40"
            disabled={disabled || value === setting.recommended}
            onClick={onReset}
            title={`Recommended: ${setting.recommended}`}
          >
            use {setting.recommended}
          </button>
        </div>
      </div>
    </li>
  );
}
