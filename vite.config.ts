import { defineConfig, splitVendorChunkPlugin, type PluginOption } from 'vite';
import react from "@vitejs/plugin-react";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import { visualizer } from "rollup-plugin-visualizer";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    TanStackRouterVite(),
    visualizer({
      filename: 'vite/stats.html',
      gzipSize: true,
    }) as PluginOption,
    splitVendorChunkPlugin(),
  ],
  server: {
    host: "0.0.0.0",
    proxy: {
      "/api": {
        target: "http://localhost:3000",
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
