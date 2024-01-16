import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    exclude: ["./src/integration/**", "./src/routes/**"],
    reporters: ["html", "verbose"],
    outputFile: "./.vitest/html",
    alias: {
      "@/": new URL("./src/", import.meta.url).pathname,
    },
  },
});
