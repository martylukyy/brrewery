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
    <div className="flex items-center gap-2 text-muted-foreground">
      <Label htmlFor={id}>Range</Label>
      <Select value={value} onValueChange={(next) => onChange(next as VnstatRangeId)}>
        <SelectTrigger id={id} aria-label="vnStat range">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {VNSTAT_RANGE_OPTIONS.map((option) => (
            <SelectItem key={option.id} value={option.id}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
