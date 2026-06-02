import {
  CHART_INTERVAL_OPTIONS,
  type ChartIntervalId,
} from "@/lib/chart-interval";

type Props = {
  value: ChartIntervalId;
  onChange: (id: ChartIntervalId) => void;
  id?: string;
};

export function ChartIntervalSelect({ value, onChange, id = "chart-interval" }: Props) {
  return (
    <label htmlFor={id} className="flex items-center gap-2 text-xs text-zinc-500">
      <span>Time range</span>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value as ChartIntervalId)}
        className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1 text-xs text-zinc-200"
        aria-label="Chart time range"
      >
        {CHART_INTERVAL_OPTIONS.map((option) => (
          <option key={option.id} value={option.id}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}
