import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    reporters: ["html", "verbose"],
    outputFile: "./.vitest/html",
  },
  resolve: {
    mainFields: ["module"],
  },
});
