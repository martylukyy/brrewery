/** Megabit/s and gigabit/s as bytes/s (decimal bits: 1 Mbit = 1e6 bit/s). */
export const NETWORK_SCALE_OPTIONS = [
  { id: "100mbit", label: "100 Mbit/s", maxBytesPerSec: 100e6 / 8 },
  { id: "1gbit", label: "1 Gbit/s", maxBytesPerSec: 1e9 / 8 },
  { id: "10gbit", label: "10 Gbit/s", maxBytesPerSec: 10e9 / 8 },
] as const;

export type NetworkScaleId = (typeof NETWORK_SCALE_OPTIONS)[number]["id"];

export const DEFAULT_NETWORK_SCALE: NetworkScaleId = "1gbit";

export function networkScaleMaxBytes(id: NetworkScaleId): number {
  const option = NETWORK_SCALE_OPTIONS.find((o) => o.id === id);
  return option?.maxBytesPerSec ?? NETWORK_SCALE_OPTIONS[1].maxBytesPerSec;
}
