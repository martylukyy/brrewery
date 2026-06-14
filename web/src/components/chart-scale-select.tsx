import { Label } from "@/components/ui/label";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";

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
      <NativeSelect
        id={id}
        size="sm"
        value={value}
        onChange={(event) => onChange(event.target.value as T)}
        aria-label={ariaLabel}
      >
        {options.map((option) => (
          <NativeSelectOption key={option.id} value={option.id}>
            {option.label}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </div>
  );
}
