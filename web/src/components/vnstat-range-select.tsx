import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { VNSTAT_RANGE_OPTIONS, type VnstatRangeId } from "@/lib/vnstat-range";

type Props = {
  value: VnstatRangeId;
  onChange: (id: VnstatRangeId) => void;
  id?: string;
};

export function VnstatRangeSelect({ value, onChange, id = "vnstat-range" }: Props) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <Label htmlFor={id} className="text-xs text-muted-foreground">
        Range
      </Label>
      <Select
        value={String(value)}
        onValueChange={(next) => onChange(Number(next) as VnstatRangeId)}
      >
        <SelectTrigger id={id} size="sm" className="text-xs" aria-label="vnStat range">
          <SelectValue />
        </SelectTrigger>
        <SelectContent position="popper" align="end">
          {VNSTAT_RANGE_OPTIONS.map((option) => (
            <SelectItem key={option.id} value={String(option.id)} className="text-xs">
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
