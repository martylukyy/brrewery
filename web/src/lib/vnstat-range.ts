export const VNSTAT_RANGE_OPTIONS = [
  { id: "months", label: "Last 12 months" },
  { id: "days", label: "Last 30 days" },
  { id: "top10", label: "Top 10 days overall" },
] as const;

export type VnstatRangeId = (typeof VNSTAT_RANGE_OPTIONS)[number]["id"];

export const DEFAULT_VNSTAT_RANGE: VnstatRangeId = "days";

export function isVnstatRangeId(value: unknown): value is VnstatRangeId {
  return VNSTAT_RANGE_OPTIONS.some((option) => option.id === value);
}
