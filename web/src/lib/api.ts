const API_BASE = "/api/v1";

export type ErrorBody = {
  error: string;
};

export class ApiError extends Error {
  status: number;
  path: string;

  constructor(message: string, status: number, path = "") {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.path = path;
  }
}

// Endpoints where a 401 means "wrong credentials" rather than an expired
// session. A failure here must not sign the user out: a mistyped login or a
// password-confirmation prompt should keep the user where they are.
const CREDENTIAL_PATHS = ["/auth/login", "/auth/verify-password"];

// isSessionExpired reports whether an error indicates the session cookie is no
// longer valid (invalid or too old), as opposed to a deliberately rejected
// credential check. Used to route the user back to the login page.
export function isSessionExpired(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    error.status === 401 &&
    !CREDENTIAL_PATHS.includes(error.path)
  );
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
    throw new ApiError(await readErrorMessage(res), res.status, path);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

export type LoginRequest = {
  username: string;
  password: string;
  remember_me: boolean;
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

export type AppStatus = {
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

export type AppListResponse = {
  apps: AppStatus[];
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

// verifyPassword checks a candidate password against the signed-in user's
// account. It resolves on a match and throws ApiError (401) on a mismatch.
export function verifyPassword(password: string): Promise<void> {
  return apiFetch<void>("/auth/verify-password", {
    method: "POST",
    body: JSON.stringify({ password }),
  });
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
    throw new ApiError(await readErrorMessage(res), res.status, "/version");
  }

  return (await res.json()) as VersionInfo;
}

export function listApps() {
  return apiFetch<AppListResponse>("/apps");
}

export type JobAction = "install" | "upgrade" | "remove";

export type JobStatus = "queued" | "running" | "succeeded" | "failed";

export type AppJob = {
  id: string;
  app_id: string;
  action: JobAction;
  status: JobStatus;
  error?: string;
  started_at: string;
  finished_at?: string;
};

export type AppJobRequest = {
  extra_vars?: Record<string, string>;
};

export type AppJobResponse = {
  job_id: string;
};

export type JobLogsResponse = {
  lines: string[];
};

export function startAppJob(id: string, action: JobAction, body: AppJobRequest = {}) {
  return apiFetch<AppJobResponse>(`/apps/${encodeURIComponent(id)}/${action}`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function installApp(id: string, body: AppJobRequest = {}) {
  return startAppJob(id, "install", body);
}

export function upgradeApp(id: string, body: AppJobRequest = {}) {
  return startAppJob(id, "upgrade", body);
}

export function removeApp(id: string, body: AppJobRequest = {}) {
  return startAppJob(id, "remove", body);
}

export function getJob(id: string) {
  return apiFetch<AppJob>(`/jobs/${encodeURIComponent(id)}`);
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

export type SysctlKind = "integer" | "integer_list" | "enum";

export type SysctlParam = {
  key: string;
  label: string;
  description: string;
  group: string;
  kind: SysctlKind | string;
  recommended: string;
  unit?: string;
  min?: number;
  max?: number;
  fields?: number;
  choices?: string[];
};

export type SysctlSetting = SysctlParam & {
  value: string;
  available: boolean;
};

export type SysctlReport = {
  settings: SysctlSetting[];
  writable: boolean;
};

export function getSysctl() {
  return apiFetch<SysctlReport>("/system/sysctl");
}

export type ApplySysctlRequest = {
  values: Record<string, string>;
  password: string;
};

// applySysctl persists the given kernel parameters and returns the refreshed
// report with the new live values.
export function applySysctl(body: ApplySysctlRequest) {
  return apiFetch<SysctlReport>("/system/sysctl", {
    method: "POST",
    body: JSON.stringify(body),
  });
}
