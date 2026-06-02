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
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-4">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-sm font-semibold text-zinc-200">{title}</h2>
          {subtitle && <p className="text-xs text-zinc-500">{subtitle}</p>}
          {waiting && (
            <p className="mt-1 text-xs text-zinc-600">
              Collecting samples… (updates every {pollSeconds}s)
            </p>
          )}
        </div>
        {action}
      </div>
      {children}
    </div>
  );
}
