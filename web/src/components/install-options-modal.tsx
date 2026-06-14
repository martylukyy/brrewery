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
import type { InstallOption, AppStatus } from "@/lib/api";

const VERSION_KEY = "qbittorrent_version";
const BRANCH_KEY = "libtorrent_branch";
const PATCH_KEY = "libtorrent_patch";
const MAX_PATCH_BYTES = 512 * 1024;

type Props = {
  appIds: string[];
  apps: AppStatus[];
  onClose: () => void;
  onConfirm: (extraVars: Record<string, string>) => void;
};

/**
 * requiredInstallOptions returns the install options of the first selected
 * app that declares any. Only qBittorrent does today.
 */
export function requiredInstallOptions(apps: AppStatus[], appIds: string[]): InstallOption[] {
  for (const id of appIds) {
    const app = apps.find((entry) => entry.id === id);
    if (app?.install_options?.length) {
      return app.install_options;
    }
  }
  return [];
}

function optionByKey(options: InstallOption[], key: string): InstallOption | undefined {
  return options.find((option) => option.key === key);
}

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error ?? new Error("Failed to read file"));
    reader.onload = () => {
      const result = String(reader.result ?? "");
      const comma = result.indexOf(",");
      resolve(comma >= 0 ? result.slice(comma + 1) : result);
    };
    reader.readAsDataURL(file);
  });
}

export function InstallOptionsModal({ appIds, apps, onClose, onConfirm }: Props) {
  const options = useMemo(() => requiredInstallOptions(apps, appIds), [apps, appIds]);
  const versionOption = optionByKey(options, VERSION_KEY);
  const branchOption = optionByKey(options, BRANCH_KEY);

  const appNames = appIds
    .map((id) => apps.find((app) => app.id === id)?.name ?? id)
    .join(", ");

  const [step, setStep] = useState<"version" | "libtorrent">("version");
  const [version, setVersion] = useState<string>(() => versionOption?.choices?.[0]?.value ?? "");
  const [branch, setBranch] = useState<string>(() => branchOption?.choices?.[0]?.value ?? "");
  const [patch, setPatch] = useState<{ name: string; base64: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const branchVisible = useMemo(() => {
    if (!branchOption?.when?.one_of) {
      return Boolean(branchOption);
    }
    return branchOption.when.one_of.includes(version);
  }, [branchOption, version]);

  async function handleFileChange(event: React.ChangeEvent<HTMLInputElement>) {
    setError(null);
    const file = event.target.files?.[0];
    if (!file) {
      setPatch(null);
      return;
    }
    if (file.size > MAX_PATCH_BYTES) {
      setPatch(null);
      setError("Patch file is too large (max 512 KiB).");
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
      return;
    }
    try {
      const base64 = await readFileAsBase64(file);
      setPatch({ name: file.name, base64 });
    } catch {
      setError("Could not read the selected patch file.");
    }
  }

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);

    if (step === "version") {
      if (!version) {
        setError("Select a qBittorrent version.");
        return;
      }
      setStep("libtorrent");
      return;
    }

    const extraVars: Record<string, string> = { [VERSION_KEY]: version };
    if (branchVisible && branch) {
      extraVars[BRANCH_KEY] = branch;
    }
    if (patch) {
      extraVars[PATCH_KEY] = patch.base64;
    }
    onConfirm(extraVars);
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[90vh] flex-col gap-0 p-0 sm:max-w-md">
        <form className="flex min-h-0 flex-1 flex-col" onSubmit={handleSubmit}>
          <DialogHeader className="gap-1 border-b border-border px-5 py-4">
            <DialogTitle className="text-base">
              {step === "version" ? "Choose qBittorrent version" : "libtorrent options"}
            </DialogTitle>
            <DialogDescription>
              {step === "version"
                ? `Select the qBittorrent release to build for ${appNames}.`
                : "Pick the libtorrent line and optionally supply a custom patch."}
            </DialogDescription>
          </DialogHeader>

          <div className="scrollbar-zinc min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
            {step === "version" && (
              <fieldset className="space-y-2">
                <legend className="sr-only">qBittorrent version</legend>
                {versionOption?.choices?.map((choice) => (
                  <label
                    key={choice.value}
                    className="flex cursor-pointer items-center gap-3 rounded-md border border-border px-3 py-2 hover:bg-accent/50"
                  >
                    <input
                      type="radio"
                      name="qbittorrent-version"
                      value={choice.value}
                      checked={version === choice.value}
                      onChange={() => setVersion(choice.value)}
                    />
                    <span className="text-sm text-foreground">{choice.label}</span>
                  </label>
                ))}
              </fieldset>
            )}

            {step === "libtorrent" && (
              <>
                {branchVisible && (
                  <fieldset className="space-y-2">
                    <legend className="mb-1 block text-sm text-foreground">libtorrent version</legend>
                    {branchOption?.choices?.map((choice) => (
                      <label
                        key={choice.value}
                        className="flex cursor-pointer items-center gap-3 rounded-md border border-border px-3 py-2 hover:bg-accent/50"
                      >
                        <input
                          type="radio"
                          name="libtorrent-branch"
                          value={choice.value}
                          checked={branch === choice.value}
                          onChange={() => setBranch(choice.value)}
                        />
                        <span className="text-sm text-foreground">{choice.label}</span>
                      </label>
                    ))}
                  </fieldset>
                )}

                <div className="space-y-1">
                  <Label htmlFor="libtorrent-patch">Custom libtorrent patch (optional)</Label>
                  <Input
                    ref={fileInputRef}
                    id="libtorrent-patch"
                    type="file"
                    accept=".patch,.diff,text/plain"
                    onChange={handleFileChange}
                  />
                  <span className="block text-xs text-muted-foreground">
                    Leave empty to use brrewery&apos;s default performance patch. Applied to this build only; not saved.
                  </span>
                  {patch && <span className="block text-xs text-emerald-400">Selected {patch.name}</span>}
                </div>
              </>
            )}

            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>

          <DialogFooter className="border-t border-border px-5 py-4 sm:justify-between">
            <Button
              type="button"
              variant="outline"
              onClick={step === "libtorrent" ? () => setStep("version") : onClose}
            >
              {step === "libtorrent" ? "Back" : "Cancel"}
            </Button>
            <Button type="submit">
              {step === "version" ? "Continue" : "Start install"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
