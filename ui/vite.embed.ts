import { defineConfig, type PluginOption } from "vite";
import react from "@vitejs/plugin-react";
import { visualizer } from "rollup-plugin-visualizer";
import path from "path";
import cssInjectedByJsPlugin from "vite-plugin-css-injected-by-js";
import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';

export default defineConfig({
  plugins: [
    react(),
    cssInjectedByJsPlugin(),
    visualizer({
      filename: ".vite-stats/stats-embed.html",
      gzipSize: true,
    }) as PluginOption,
  ],
  css: {
    postcss: {
      plugins: [
        tailwindcss({ config: 'ui/tailwind.config.js' }),
        autoprefixer,
      ],
    }
  },
  build: {
    outDir: path.resolve(__dirname, "../dist/embed"),
    copyPublicDir: false,
    lib: {
      name: "shaper",
      entry: path.resolve(__dirname, "src/embed.tsx"),
      formats: ["umd"],
      fileName: () => "shaper.js",
    },
    sourcemap: false,
  },
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
});
