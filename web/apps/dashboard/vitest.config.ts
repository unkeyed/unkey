import { defineConfig } from "vitest/config";

export default defineConfig({
  // Workspace packages resolve their own tsconfig, which still asks for the
  // classic runtime — their JSX then reaches for an undeclared `React`.
  esbuild: { jsx: "automatic" },
  test: {
    environment: "jsdom",
    alias: { "@/": new URL("./", import.meta.url).pathname },
  },
});
