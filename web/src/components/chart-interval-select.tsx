import {
  CHART_INTERVAL_OPTIONS,
  type ChartIntervalId,
} from "@/lib/chart-interval";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select";

type Props = {
  value: ChartIntervalId;
  onChange: (id: ChartIntervalId) => void;
  id?: string;
};

export function ChartIntervalSelect({ value, onChange, id = "chart-interval" }: Props) {
  return (
    <label htmlFor={id} className="flex items-center gap-2 text-xs text-muted-foreground">
      <span>Time range</span>
      <NativeSelect
        id={id}
        size="sm"
        value={value}
        onChange={(e) => onChange(e.target.value as ChartIntervalId)}
        aria-label="Chart time range"
      >
        {CHART_INTERVAL_OPTIONS.map((option) => (
          <NativeSelectOption key={option.id} value={option.id}>
            {option.label}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </label>
  );
}
