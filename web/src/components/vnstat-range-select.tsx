export type VnstatRangeId = "days" | "months" | "top10";

const VNSTAT_RANGE_OPTIONS = [
  { id: "months" as const, label: "Last 12 months" },
  { id: "days" as const, label: "Last 30 days" },
  { id: "top10" as const, label: "Top 10 days overall" },
];

type Props = {
  value: VnstatRangeId;
  onChange: (id: VnstatRangeId) => void;
  id?: string;
};

export function VnstatRangeSelect({ value, onChange, id = "vnstat-range" }: Props) {
  return (
    <label htmlFor={id} className="flex items-center gap-2 text-xs text-zinc-500">
      <span>Range</span>
      <select
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value as VnstatRangeId)}
        className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1 text-xs text-zinc-200"
        aria-label="vnStat range"
      >
        {VNSTAT_RANGE_OPTIONS.map((option) => (
          <option key={option.id} value={option.id}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}
