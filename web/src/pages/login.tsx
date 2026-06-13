import { useState } from "react";

import { useAuth } from "@/hooks/use-auth";

export function LoginPage() {
  const { login } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [rememberMe, setRememberMe] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await login.mutateAsync({ username, password, remember_me: rememberMe });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    }
  }

  return (
    <div className="relative isolate flex min-h-screen flex-col items-center justify-center bg-zinc-950 px-4">
      <svg
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 -z-10 h-full w-full text-zinc-900/25"
      >
        <defs>
          <pattern id="brick-wall" width={42} height={44} patternUnits="userSpaceOnUse">
            <path
              d="M0 0h42v44H0V0zm1 1h40v20H1V1zM0 23h20v20H0V23zm22 0h20v20H22V23z"
              fill="currentColor"
              fillRule="evenodd"
            />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#brick-wall)" />
      </svg>
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-md rounded-lg border border-zinc-800 bg-zinc-900 p-6 shadow-[0_0_32px_0_var(--color-white)]/5 animate-glow motion-reduce:animate-none"
      >
        <div className="flex flex-col items-center mb-4">
          <img src="/brrewery.png" className="size-16"/>
          <h1 className="text-xl font-semibold">brrewery</h1>
          <p className="text-sm text-zinc-500">seedbox management suite.</p>
        </div>
   
        <div className="space-y-4">
          <label className="block text-sm text-zinc-300">
            Username
            <input
              className="mt-1 w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              required
            />
          </label>
          <label className="block text-sm text-zinc-300">
            Password
            <input
              type="password"
              className="mt-1 w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              required
            />
          </label>
          <label className="flex items-center gap-2 text-sm text-zinc-300">
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-zinc-700 bg-zinc-950 accent-blue-600"
              checked={rememberMe}
              onChange={(e) => setRememberMe(e.target.checked)}
            />
            Remember me
          </label>
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            type="submit"
            className="w-full rounded-md px-4 py-2 font-medium bg-blue-600 hover:bg-blue-700"
            disabled={login.isPending}
          >
            {login.isPending ? "Signing in…" : "Sign in"}
          </button>
        </div>
      </form>
    </div>
  );
}
