import { Label } from "@/components/ui/label";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
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
      <NativeSelect
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value as VnstatRangeId)}
        aria-label="vnStat range"
      >
        {VNSTAT_RANGE_OPTIONS.map((option) => (
          <NativeSelectOption key={option.id} value={option.id}>
            {option.label}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </div>
  );
}
