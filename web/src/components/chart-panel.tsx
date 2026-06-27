import type { ReactNode } from "react";

type Props = {
  title: string;
  subtitle?: string;
  waiting: boolean;
  pollSeconds: number;
  action?: ReactNode;
  children: ReactNode;
};

export function ChartPanel({
  title,
  subtitle,
  waiting,
  pollSeconds,
  action,
  children,
}: Props) {
  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card p-4">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-sm font-semibold text-foreground">{title}</h2>
          {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
          {waiting && (
            <p className="mt-1 text-xs text-muted-foreground">
              Collecting samples… (updates every {pollSeconds}s)
            </p>
          )}
        </div>
        {action}
      </div>
      <div className="flex min-h-0 flex-1 flex-col">{children}</div>
    </div>
  );
}
