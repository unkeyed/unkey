import { defineConfig } from "vitest/config";
export default defineConfig({
  test: {
    exclude: [],
    pool: "threads",
    poolOptions: {
      threads: {
        minThreads: 1,
        maxThreads: 2,
      },
    },
  },
});
