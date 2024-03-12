import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    dir: "./src/routes",
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
    // starting the worker takes a bit of time
    testTimeout: 60_000,
  },
});
