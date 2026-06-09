/** Single localStorage key holding every persisted UI setting as one JSON object. */
export const SETTINGS_STORAGE_KEY = "brrewery_settings";

type SettingsRecord = Record<string, unknown>;

function readAll(): SettingsRecord {
  try {
    const raw = window.localStorage.getItem(SETTINGS_STORAGE_KEY);
    if (raw === null) {
      return {};
    }
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return {};
    }
    return parsed as SettingsRecord;
  } catch {
    return {};
  }
}

/**
 * Reads one named setting from the shared object. Pass `isValid` to reject stale
 * or corrupt values (e.g. an option id that no longer exists) and fall back to
 * `defaultValue`.
 */
export function readSetting<T>(
  field: string,
  defaultValue: T,
  isValid?: (value: unknown) => value is T,
): T {
  const stored = readAll()[field];
  if (stored === undefined) {
    return defaultValue;
  }
  if (isValid && !isValid(stored)) {
    return defaultValue;
  }
  return stored as T;
}

/** Merges one named setting into the shared object, leaving other fields intact. */
export function writeSetting(field: string, value: unknown): void {
  try {
    const next = { ...readAll(), [field]: value };
    window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(next));
  } catch {
    // Ignore write failures (quota exceeded, private mode, storage disabled).
  }
}
