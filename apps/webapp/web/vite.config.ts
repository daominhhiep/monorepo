import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Vite proxies Connect-RPC traffic + auth endpoints to the BFF in dev,
// so the SPA can run on a separate port without CORS.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/apps.webapp.v1.APIService": {
        target: "http://localhost:8082",
        changeOrigin: false,
      },
      "/healthz": "http://localhost:8082",
    },
  },
});
