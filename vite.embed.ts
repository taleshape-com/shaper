import { defineConfig, type PluginOption } from 'vite';
import react from "@vitejs/plugin-react";
import { visualizer } from "rollup-plugin-visualizer";
import path from 'path';
import cssInjectedByJsPlugin from 'vite-plugin-css-injected-by-js';

export default defineConfig({
  plugins: [
    react(),
    cssInjectedByJsPlugin(),
    visualizer() as PluginOption,
  ],
  build: {
    outDir: "dist/embed",
    copyPublicDir: false,
    lib: {
      name: "shaper",
      entry: path.resolve(__dirname, "src/embed.tsx"),
      formats: ["umd"],
      fileName: () => "shaper.js",
    },
    sourcemap: true,
  },
  define: {
    'process.env.NODE_ENV': JSON.stringify('production')
  }
})
