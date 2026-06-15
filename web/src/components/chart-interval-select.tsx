import {
  CHART_INTERVAL_OPTIONS,
  type ChartIntervalId,
} from "@/lib/chart-interval";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

type Props = {
  value: ChartIntervalId;
  onChange: (id: ChartIntervalId) => void;
  id?: string;
};

export function ChartIntervalSelect({ value, onChange, id = "chart-interval" }: Props) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <Label htmlFor={id} className="text-xs text-muted-foreground">
        Time range
      </Label>
      <Select value={value} onValueChange={(next) => onChange(next as ChartIntervalId)}>
        <SelectTrigger id={id} size="sm" className="text-xs" aria-label="Chart time range">
          <SelectValue />
        </SelectTrigger>
        <SelectContent position="popper" align="end">
          {CHART_INTERVAL_OPTIONS.map((option) => (
            <SelectItem key={option.id} value={option.id} className="text-xs">
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
