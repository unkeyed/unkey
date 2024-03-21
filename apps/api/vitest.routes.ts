import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    dir: "./src/routes",
    reporters: ["html", "verbose"],
    outputFile: "./.vitest/html",
    alias: {
      "@/": new URL("./src/", import.meta.url).pathname,
    },

    // starting the worker takes a bit of time
    testTimeout: 60_000,
    teardownTimeout: 60_000,
  },
});
