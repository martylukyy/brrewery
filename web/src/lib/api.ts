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

export type InstallSecret = {
  key: string;
  label: string;
  type: "password" | string;
  verify_brrewery_password?: boolean;
  disable_password_manager?: boolean;
};

export type InstallOptionChoice = {
  value: string;
  label: string;
};

export type InstallOptionWhen = {
  key: string;
  one_of?: string[];
};

export type InstallOption = {
  key: string;
  label: string;
  type: "select" | string;
  choices?: InstallOptionChoice[];
  when?: InstallOptionWhen;
};

export type PackageStatus = {
  id: string;
  name: string;
  description: string;
  category: string;
  icon?: string;
  web_path?: string;
  install_secrets?: InstallSecret[];
  install_options?: InstallOption[];
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
  read_bytes: number;
  write_bytes: number;
  read_ops: number;
  write_ops: number;
};

export type NetworkCounters = {
  rx_bytes: number;
  tx_bytes: number;
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
  network: NetworkCounters;
};

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

export type JobAction = "install" | "upgrade" | "remove";

export type JobStatus = "queued" | "running" | "succeeded" | "failed";

export type PackageJob = {
  id: string;
  package_id: string;
  action: JobAction;
  status: JobStatus;
  error?: string;
  started_at: string;
  finished_at?: string;
};

export type PackageJobRequest = {
  extra_vars?: Record<string, string>;
};

export type PackageJobResponse = {
  job_id: string;
};

export type JobLogsResponse = {
  lines: string[];
};

export function startPackageJob(id: string, action: JobAction, body: PackageJobRequest = {}) {
  return apiFetch<PackageJobResponse>(`/packages/${encodeURIComponent(id)}/${action}`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function installPackage(id: string, body: PackageJobRequest = {}) {
  return startPackageJob(id, "install", body);
}

export function upgradePackage(id: string, body: PackageJobRequest = {}) {
  return startPackageJob(id, "upgrade", body);
}

export function removePackage(id: string, body: PackageJobRequest = {}) {
  return startPackageJob(id, "remove", body);
}

export function getJob(id: string) {
  return apiFetch<PackageJob>(`/jobs/${encodeURIComponent(id)}`);
}

export function getJobLogs(id: string) {
  return apiFetch<JobLogsResponse>(`/jobs/${encodeURIComponent(id)}/logs`);
}

export function getSystemInfo() {
  return apiFetch<SystemInfo>("/system");
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
