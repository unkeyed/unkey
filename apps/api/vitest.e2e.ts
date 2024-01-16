import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    dir: "./src/integration",
    reporters: ["html", "verbose"],
    outputFile: "./.vitest/html",
    alias: {
      "@/": new URL("./src/", import.meta.url).pathname,
    },
  },
});
