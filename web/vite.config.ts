import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const backendURL = env.VITE_BACKEND_URL ?? "http://127.0.0.1:8080";

  return {
    plugins: [react(), tailwindcss()],
    build: {
      // Emit SVGs (the app icons) as separate hashed assets served like the
      // font woff2 files, instead of inlining them as data URIs into the JS
      // chunk. Returning false disables the size-based inlining that would
      // otherwise embed the sub-4KB icons. Other assets keep the default.
      assetsInlineLimit: (file) => (file.endsWith(".svg") ? false : undefined),
    },
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      proxy: {
        "/api": {
          target: backendURL,
          changeOrigin: true,
        },
      },
    },
  };
});
