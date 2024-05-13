import { defineWorkersConfig } from "@cloudflare/vitest-pool-workers/config";

export default defineWorkersConfig({
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
    poolOptions: {
      workers: {
        singleWorker: true, // fails horribly without
        wrangler: {
          configPath: "wrangler.toml",
        },
      },
    },
  },
});
