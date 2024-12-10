import { defineConfig } from "vitest/config";
export default defineConfig({
  test: {
    exclude: [],
    bail: 1,
    pool: "threads",
    poolOptions: {
      threads: {
        singleThread: true,
      },
    },
  },
});
