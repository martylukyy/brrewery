export function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(1)} ${units[unit]}`;
}

export function formatRate(bytesPerSec: number): string {
  if (bytesPerSec < 1024) {
    return `${bytesPerSec.toFixed(0)} B/s`;
  }
  if (bytesPerSec < 1024 * 1024) {
    return `${(bytesPerSec / 1024).toFixed(1)} KiB/s`;
  }
  if (bytesPerSec < 1024 * 1024 * 1024) {
    return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MiB/s`;
  }
  return `${(bytesPerSec / (1024 * 1024 * 1024)).toFixed(2)} GiB/s`;
}

export function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}
