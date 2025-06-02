import { defineConfig } from "vitest/config";

// biome-ignore lint/style/noDefaultExport: required by vitests
export default defineConfig({
  test: {
    dir: "./src/benchmarks",
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
