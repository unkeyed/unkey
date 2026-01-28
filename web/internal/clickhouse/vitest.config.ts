import { defineConfig } from "vitest/config";
export default defineConfig({
  test: {
    exclude: ["**/node_modules/**", "**/dist/**"],
    bail: 1,
    pool: "threads",
    poolOptions: {
      threads: {
        singleThread: true,
      },
    },
  },
});
