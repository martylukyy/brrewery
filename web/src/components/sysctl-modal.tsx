import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useRef, useState } from "react";

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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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

  // Apply, Upload patch, and Apply recommended all converge on the password
  // prompt; on confirm the current panel values are applied (the latter two
  // having merged their values in first).
  function requestApply() {
    setUploadError(null);
    setPromptOpen(true);
  }

  // Load every readable parameter's recommended value into the panel, then apply.
  function applyRecommended() {
    const recommended: Record<string, string> = {};
    for (const setting of report?.settings ?? []) {
      if (setting.available) {
        recommended[setting.key] = setting.recommended;
      }
    }
    setApplied(false);
    setUploadError(null);
    setEdits((current) => ({ ...current, ...recommended }));
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
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        showCloseButton={false}
        className="flex max-h-[90vh] flex-col gap-0 p-0 sm:max-w-3/4"
      >
        <DialogHeader className="gap-1 border-b border-border px-5 py-4">
          <DialogTitle className="text-base">Tune sysctl parameters</DialogTitle>
          <DialogDescription>
            Adjust kernel parameters for network and storage throughput on this host.
          </DialogDescription>
        </DialogHeader>

        <div className="scrollbar-zinc min-h-0 flex-1 overflow-y-auto px-5 py-3">
          {query.isLoading && (
            <p className="py-6 text-center text-sm text-muted-foreground">Loading parameters…</p>
          )}
          {query.isError && (
            <p className="py-6 text-center text-sm text-destructive">{(query.error as Error).message}</p>
          )}

          {report && !writable && (
            <p className="mb-3 rounded-md border border-amber-900/60 bg-amber-950/30 px-3 py-2 text-xs text-amber-500">
              Applying changes is not supported on this platform. Values are shown for reference only.
            </p>
          )}

          {groups.map((group) => (
            <section key={group.name} className="mb-5">
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
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

        <DialogFooter className="border-t border-border px-5 py-4 sm:justify-between">
          <input
            ref={fileInputRef}
            type="file"
            accept=".conf,.txt,text/plain"
            className="hidden"
            aria-hidden="true"
            tabIndex={-1}
            onChange={handlePatchFile}
          />

          <p className="min-w-0 flex-1 truncate text-sm">
            {uploadError && <span className="text-destructive">{uploadError}</span>}
            {apply.isError && (
              <span className="text-destructive">{(apply.error as Error).message}</span>
            )}
            {applied && <span className="text-emerald-400">Settings applied.</span>}
          </p>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              variant="outline"
              disabled={!writable || apply.isPending}
              onClick={applyRecommended}
            >
              Apply recommended
            </Button>
            <Button
              variant="outline"
              disabled={!writable || apply.isPending}
              onClick={() => fileInputRef.current?.click()}
            >
              Upload patch
            </Button>
            <Button disabled={!writable || apply.isPending} onClick={requestApply}>
              {apply.isPending ? "Applying…" : "Apply"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>

      {promptOpen && (
        <SysctlPasswordPrompt onCancel={() => setPromptOpen(false)} onConfirm={confirmPassword} />
      )}
    </Dialog>
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

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent showCloseButton={false} className="gap-0 p-0 sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader className="gap-1 border-b border-border px-5 py-4">
            <DialogTitle className="text-base">Confirm your password</DialogTitle>
            <DialogDescription>
              Enter your account password to apply the sysctl changes.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 px-5 py-4">
            <div className="space-y-1">
              <Label htmlFor="sysctl-password">Account password</Label>
              <Input
                id="sysctl-password"
                type="password"
                value={password}
                name="password"
                autoComplete="current-password"
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>

          <DialogFooter className="border-t border-border px-5 py-4">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={verifying}>
              {verifying ? "Verifying…" : "Continue"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
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
    <li className="flex flex-col gap-1 rounded-md px-2 py-2 hover:bg-accent/50 sm:flex-row sm:items-start sm:gap-4">
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <span className="text-sm font-medium text-foreground">{setting.label}</span>
          {setting.unit && (
            <span className="text-xs text-muted-foreground">in {setting.unit}</span>
          )}
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground">{setting.description}</p>
        <p className="mt-0.5 font-mono text-xs text-muted-foreground">{setting.key}</p>
      </div>

      <div className="flex shrink-0 flex-col items-stretch gap-1 sm:w-96">
        {setting.kind === "enum" && setting.choices ? (
          <Select
            value={value || undefined}
            disabled={disabled}
            onValueChange={onChange}
          >
            <SelectTrigger className="w-full" aria-label={setting.label}>
              <SelectValue placeholder="—" />
            </SelectTrigger>
            <SelectContent>
              {/* Surface the live value even when it is outside the recommended choices. */}
              {value && !setting.choices.includes(value) && (
                <SelectItem value={value}>{value}</SelectItem>
              )}
              {setting.choices.map((choice) => (
                <SelectItem key={choice} value={choice}>
                  {choice}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <Input
            type="text"
            inputMode={setting.kind === "integer" ? "numeric" : "text"}
            aria-label={setting.label}
            placeholder={setting.available ? undefined : "unavailable"}
            value={value}
            disabled={disabled}
            onChange={(event) => onChange(event.target.value)}
          />
        )}
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span className="truncate" title={`Current: ${current}`}>
            now: {current || "—"}
          </span>
          <Button
            type="button"
            variant="link"
            size="xs"
            className="h-auto px-1"
            disabled={disabled || value === setting.recommended}
            onClick={onReset}
            title={`Recommended: ${setting.recommended}`}
          >
            use {setting.recommended}
          </Button>
        </div>
      </div>
    </li>
  );
}
