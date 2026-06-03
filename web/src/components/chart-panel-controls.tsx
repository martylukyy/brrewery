import type { ReactNode } from "react";

type Props = {
  timeRange: ReactNode;
  leading?: ReactNode;
};

export function ChartPanelControls({ timeRange, leading }: Props) {
  return (
    <div className="flex shrink-0 flex-wrap content-start items-start justify-end gap-3">
      {leading}
      {timeRange}
    </div>
  );
}
