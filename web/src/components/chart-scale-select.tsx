type ScaleOption<T extends string> = {
  id: T;
  label: string;
};

type Props<T extends string> = {
  id: string;
  value: T;
  options: readonly ScaleOption<T>[];
  onChange: (value: T) => void;
  ariaLabel: string;
};

export function ChartScaleSelect<T extends string>({
  id,
  value,
  options,
  onChange,
  ariaLabel,
}: Props<T>) {
  return (
    <label htmlFor={id} className="flex items-center gap-2 text-xs text-zinc-500">
      <span>Scale</span>
      <select
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value as T)}
        className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1 text-xs text-zinc-200"
        aria-label={ariaLabel}
      >
        {options.map((option) => (
          <option key={option.id} value={option.id}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}
