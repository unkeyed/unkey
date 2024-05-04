import path from "node:path";
import { configDefaults, defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: [...configDefaults.include, "**/__tests__/**/*.{ts,tsx}"],
    deps: {
      moduleDirectories: [
        "node_modules",
        path.resolve("../node_modules/@redwoodjs/vite/dist/middleware"),
      ],
    },
  },
});
