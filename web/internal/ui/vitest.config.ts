import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "jsdom",
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    alias: { "@/": new URL("./src/", import.meta.url).pathname },
  },
});
