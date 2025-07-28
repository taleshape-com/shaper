import path from 'path';
import { defineConfig, type PluginOption } from 'vite';
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { visualizer } from "rollup-plugin-visualizer";

// https://vite.dev/config/
export default defineConfig({
  build: {
    modulePreload: false,
    outDir: path.join(__dirname, "../dist"),
    emptyOutDir: true,
  },
  plugins: [
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
      routesDirectory: 'ui/src/routes',
      generatedRouteTree: 'ui/src/routeTree.gen.ts',
    }),
    react(),
    visualizer({
      filename: '.vite-stats/stats.html',
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
      "/embed": {
        target: "http://localhost:5454",
        changeOrigin: true,
        secure: false,
      },
      "/view": {
        target: "http://localhost:5454",
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
