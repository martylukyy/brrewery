import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

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
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <Label htmlFor={id} className="text-xs text-muted-foreground">
        Scale
      </Label>
      <Select value={value} onValueChange={(next) => onChange(next as T)}>
        <SelectTrigger id={id} size="sm" className="text-xs" aria-label={ariaLabel}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent position="popper" align="end">
          {options.map((option) => (
            <SelectItem key={option.id} value={option.id} className="text-xs">
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
