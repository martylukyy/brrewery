import { useCallback, useState } from "react";

import { readSetting, writeSetting } from "@/lib/settings";

/**
 * useState for a single named UI setting persisted in the shared
 * `brrewery_settings` localStorage object.
 *
 * The stored value is read once on mount; pass `isValid` to reject stale or
 * corrupt values and fall back to `defaultValue`. Writes merge into the shared
 * object in the setter, so storage stays in sync with user-driven changes
 * without an effect and without clobbering other settings.
 */
export function useSetting<T>(
  field: string,
  defaultValue: T,
  isValid?: (value: unknown) => value is T,
): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => readSetting(field, defaultValue, isValid));

  const setStoredValue = useCallback(
    (next: T) => {
      setValue(next);
      writeSetting(field, next);
    },
    [field],
  );

  return [value, setStoredValue];
}
