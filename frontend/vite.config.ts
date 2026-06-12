import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import svgr from "vite-plugin-svgr";

export default defineConfig({
  server: {
    host: "0.0.0.0",
    port: 5173,
    strictPort: true,
    allowedHosts: [
      "discom.spbgu.90.188.89.63.nip.io",
      "discom.spbgu.localhost",
    ],
    hmr: {
      clientPort: 8443,
      protocol: "wss",
    },
    watch: {
      usePolling: true,
      interval: 300,
    },
  },
  plugins: [react(), svgr()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
