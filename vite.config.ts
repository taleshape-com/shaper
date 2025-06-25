import { defineConfig, type PluginOption } from 'vite';
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { visualizer } from "rollup-plugin-visualizer";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    visualizer({
      filename: 'vite/stats.html',
      gzipSize: true,
    }) as PluginOption,
  ],
  server: {
    host: "0.0.0.0",
    port: 5453,
    proxy: {
      "/api": {
        target: "http://localhost:5454",
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
