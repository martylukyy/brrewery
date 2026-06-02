const API_BASE = "/api/v1";

export type ErrorBody = {
  error: string;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function readErrorMessage(res: Response): Promise<string> {
  let message = res.statusText;
  try {
    const body = (await res.json()) as ErrorBody;
    if (body.error) {
      message = body.error;
    }
  } catch {
    // ignore parse errors
  }
  return message;
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
    ...init,
  });

  if (!res.ok) {
    throw new ApiError(await readErrorMessage(res), res.status);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

export type LoginRequest = {
  username: string;
  password: string;
};

export type LoginResponse = {
  username: string;
};

export type VersionInfo = {
  version: string;
  commit: string;
  date: string;
};

export type PackageStatus = {
  id: string;
  name: string;
  description: string;
  category: string;
  installed: boolean;
  dependencies_satisfied: boolean;
};

export type PackageListResponse = {
  packages: PackageStatus[];
};

export type LoadAvg = {
  "1m": number;
  "5m": number;
  "15m": number;
};

export type SystemMemory = {
  total_bytes: number;
  available_bytes: number;
  used_bytes: number;
  used_percent: number;
};

export type SystemDisk = {
  mount: string;
  total_bytes: number;
  used_bytes: number;
  available_bytes: number;
  used_percent: number;
  io_busy_percent?: number;
  io_read_bytes?: number;
  io_write_bytes?: number;
};

export type NetworkCounters = {
  rx_bytes: number;
  tx_bytes: number;
};

export type DiskIOCounters = {
  read_bytes: number;
  write_bytes: number;
  read_ops: number;
  write_ops: number;
};

export type SystemInfo = {
  hostname: string;
  uptime_seconds: number;
  cpu_count: number;
  cpu_name: string;
  cpu_percent: number;
  load: LoadAvg;
  memory: SystemMemory;
  disks: SystemDisk[];
  /** @deprecated Use disks. Present on older API responses. */
  disk?: SystemDisk;
  network: NetworkCounters;
  disk_io: DiskIOCounters;
};

/** @deprecated Use disks. Present on older API responses. */
export type SystemInfoRaw = SystemInfo & {
  disk_io_busy_percent?: number;
};

export function normalizeSystemInfo(info: SystemInfoRaw): SystemInfo {
  const disks = info.disks?.length ? info.disks : info.disk ? [info.disk] : [];
  const peakIO = info.disk_io_busy_percent;
  const normalized = disks.map((disk, index) => {
    if (disk.io_busy_percent != null || peakIO == null || index !== 0) {
      return disk;
    }
    return { ...disk, io_busy_percent: peakIO };
  });
  return { ...info, disks: normalized };
}

export function login(body: LoginRequest) {
  return apiFetch<LoginResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function logout() {
  return apiFetch<{ status: string }>("/auth/logout", { method: "POST" });
}

export async function checkSession(): Promise<VersionInfo | null> {
  const res = await fetch(`${API_BASE}/version`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (res.status === 401) {
    return null;
  }

  if (!res.ok) {
    throw new ApiError(await readErrorMessage(res), res.status);
  }

  return (await res.json()) as VersionInfo;
}

export function listPackages() {
  return apiFetch<PackageListResponse>("/packages");
}

export function getSystemInfo() {
  return apiFetch<SystemInfoRaw>("/system").then(normalizeSystemInfo);
}

export type TrafficPeriod = {
  label: string;
  rx_bytes: number;
  tx_bytes: number;
};

export type VnstatReport = {
  installed: boolean;
  message?: string;
  version?: string;
  days?: TrafficPeriod[];
  months?: TrafficPeriod[];
};

export function getVnstatReport() {
  return apiFetch<VnstatReport>("/traffic/vnstat");
}
