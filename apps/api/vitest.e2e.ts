import { defineConfig } from "vitest/config";

export default defineConfig({
  esbuild: {
    // This allows us to use the `using` keyword
    // see https://github.com/vitejs/vite/issues/15464
    target: "es2020",
  },
  test: {
    dir: "./src/integration",
    reporters: ["html", "verbose"],
    outputFile: "./.vitest/html",
    alias: {
      "@/": new URL("./src/", import.meta.url).pathname,
    },
    pool: "threads",
    poolOptions: {
      threads: {
        singleThread: true,
      },
    },
  },
});
