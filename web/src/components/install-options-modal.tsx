import { useEffect, useMemo, useRef, useState } from "react";

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

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/60"
        aria-label="Close install options dialog"
        onClick={onClose}
      />

      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby="install-options-title"
        className="relative z-10 flex w-full max-w-md flex-col rounded-lg border border-zinc-700 bg-zinc-900 shadow-xl"
        onSubmit={handleSubmit}
      >
        <div className="border-b border-zinc-800 px-5 py-4">
          <h2 id="install-options-title" className="text-lg font-semibold text-zinc-100">
            {step === "version" ? "Choose qBittorrent version" : "libtorrent options"}
          </h2>
          <p className="mt-1 text-sm text-zinc-400">
            {step === "version"
              ? `Select the qBittorrent release to build for ${appNames}.`
              : "Pick the libtorrent line and optionally supply a custom patch."}
          </p>
        </div>

        <div className="space-y-4 px-5 py-4">
          {step === "version" && (
            <fieldset className="space-y-2">
              <legend className="sr-only">qBittorrent version</legend>
              {versionOption?.choices?.map((choice) => (
                <label key={choice.value} className="flex cursor-pointer items-center gap-3 rounded-md border border-zinc-700 px-3 py-2 hover:bg-zinc-800">
                  <input
                    type="radio"
                    name="qbittorrent-version"
                    value={choice.value}
                    checked={version === choice.value}
                    onChange={() => setVersion(choice.value)}
                  />
                  <span className="text-sm text-zinc-100">{choice.label}</span>
                </label>
              ))}
            </fieldset>
          )}

          {step === "libtorrent" && (
            <>
              {branchVisible && (
                <fieldset className="space-y-2">
                  <legend className="mb-1 block text-sm text-zinc-300">libtorrent version</legend>
                  {branchOption?.choices?.map((choice) => (
                    <label key={choice.value} className="flex cursor-pointer items-center gap-3 rounded-md border border-zinc-700 px-3 py-2 hover:bg-zinc-800">
                      <input
                        type="radio"
                        name="libtorrent-branch"
                        value={choice.value}
                        checked={branch === choice.value}
                        onChange={() => setBranch(choice.value)}
                      />
                      <span className="text-sm text-zinc-100">{choice.label}</span>
                    </label>
                  ))}
                </fieldset>
              )}

              <label className="block">
                <span className="mb-1 block text-sm text-zinc-300">Custom libtorrent patch (optional)</span>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".patch,.diff,text/plain"
                  className="block w-full text-sm text-zinc-400 file:mr-3 file:rounded-md file:border-0 file:bg-zinc-700 file:px-3 file:py-1.5 file:text-sm file:text-zinc-100 hover:file:bg-zinc-600"
                  onChange={handleFileChange}
                />
                <span className="mt-1 block text-xs text-zinc-500">
                  Leave empty to use brrewery&apos;s default performance patch. Applied to this build only; not saved.
                </span>
                {patch && <span className="mt-1 block text-xs text-emerald-400">Selected {patch.name}</span>}
              </label>
            </>
          )}

          {error && <p className="text-sm text-red-400">{error}</p>}
        </div>

        <div className="flex justify-between gap-2 border-t border-zinc-800 px-5 py-4">
          <button
            type="button"
            className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-800"
            onClick={step === "libtorrent" ? () => setStep("version") : onClose}
          >
            {step === "libtorrent" ? "Back" : "Cancel"}
          </button>
          <button
            type="submit"
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium hover:bg-blue-700"
          >
            {step === "version" ? "Continue" : "Start install"}
          </button>
        </div>
      </form>
    </div>
  );
}
