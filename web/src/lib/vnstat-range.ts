export const VNSTAT_RANGE_OPTIONS = [
  { id: 0, label: "Last 7 days", source: "days", limit: 7, sort: "recent" },
  { id: 1, label: "Last 14 days", source: "days", limit: 14, sort: "recent" },
  { id: 2, label: "Last 12 months", source: "months", limit: 12, sort: "recent" },
  { id: 3, label: "Top 10 days overall", source: "days", limit: 10, sort: "total" },
] as const;

export type VnstatRangeId = (typeof VNSTAT_RANGE_OPTIONS)[number]["id"];
type VnstatRangeSource = (typeof VNSTAT_RANGE_OPTIONS)[number]["source"];

export const DEFAULT_VNSTAT_RANGE: VnstatRangeId = 1;

export function isVnstatRangeId(value: unknown): value is VnstatRangeId {
  return VNSTAT_RANGE_OPTIONS.some((option) => option.id === value);
}

// Counts to request from the backend: the largest limit needed per source, so
// every option can be sliced client-side from a single fetch.
export function vnstatReportRequest(): { days: number; months: number } {
  const maxLimit = (source: VnstatRangeSource) =>
    VNSTAT_RANGE_OPTIONS.filter((option) => option.source === source).reduce(
      (max, option) => Math.max(max, option.limit),
      0,
    );
  return { days: maxLimit("days"), months: maxLimit("months") };
}
