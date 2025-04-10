import { defineConfig, type PluginOption } from "vite";
import react from "@vitejs/plugin-react";
import { visualizer } from "rollup-plugin-visualizer";
import path from "path";
import cssInjectedByJsPlugin from "vite-plugin-css-injected-by-js";

export default defineConfig({
  plugins: [
    react(),
    cssInjectedByJsPlugin(),
    visualizer({
      filename: "vite/stats-embed-react.html",
      gzipSize: true,
    }) as PluginOption,
  ],
  build: {
    outDir: "dist/react",
    copyPublicDir: false,
    lib: {
      name: "shaper",
      entry: path.resolve(__dirname, "src/embed-react.tsx"),
      formats: ["umd"],
      fileName: () => "shaper.js",
    },
    sourcemap: false,
    rollupOptions: {
      external: ["react", "react-dom"],
      output: {
        globals: {
          react: "React",
          "react-dom": "ReactDOM",
        },
      },
    },
  },
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
});
