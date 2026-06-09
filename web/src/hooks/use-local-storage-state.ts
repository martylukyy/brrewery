import { useCallback, useState } from "react";

function readStored<T>(
  key: string,
  defaultValue: T,
  isValid?: (value: unknown) => value is T,
): T {
  try {
    const raw = window.localStorage.getItem(key);
    if (raw === null) {
      return defaultValue;
    }
    const parsed: unknown = JSON.parse(raw);
    if (isValid && !isValid(parsed)) {
      return defaultValue;
    }
    return parsed as T;
  } catch {
    return defaultValue;
  }
}

/**
 * useState that mirrors its value to localStorage under `key`.
 *
 * The stored value is read once on mount; pass `isValid` to reject stale or
 * corrupt values (e.g. an option id that no longer exists) and fall back to
 * `defaultValue`. Writes happen in the setter, so storage stays in sync with
 * user-driven changes without an effect.
 */
export function useLocalStorageState<T>(
  key: string,
  defaultValue: T,
  isValid?: (value: unknown) => value is T,
): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => readStored(key, defaultValue, isValid));

  const setStoredValue = useCallback(
    (next: T) => {
      setValue(next);
      try {
        window.localStorage.setItem(key, JSON.stringify(next));
      } catch {
        // Ignore write failures (quota exceeded, private mode, storage disabled).
      }
    },
    [key],
  );

  return [value, setStoredValue];
}
