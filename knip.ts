import type { KnipConfig } from "knip";

const config: KnipConfig = {
  ignoreWorkspaces: [],
  ignoreDependencies: ["cz-conventional-changelog"],
  workspaces: {
    ".": {
      entry: "checkly.config.ts",
    },
    "apps/dashboard": {
      entry: ["lib/trpc/index.ts", "trpc.config.ts"],
    },
    "apps/api": {
      entry: ["**/*.test.ts", "src/pkg/testutil/*.ts", "src/worker.ts", "./vitest.*.ts"],
    },
    "internal/billing": {
      entry: ["src/index.ts", "**/*.test.ts"],
    },
    "internal/db": {
      entry: "src/index.ts",
    },
    "packages/rbac": {
      entry: ["src/index.ts", "**/*.test.ts"],
    },
    "internal/hash": {
      entry: ["src/index.ts", "**/*.test.ts"],
    },
    "internal/id": {
      entry: "src/index.ts",
    },
    "internal/keys": {
      entry: ["src/index.ts", "**/*.test.ts"],
    },
    "internal/resend": {
      entry: "src/index.ts",
    },
    "internal/vercel": {
      entry: "src/index.ts",
    },
    "packages/*": {
      entry: ["**/*.test.ts"],
    },
    "tools/bootstrap": {
      entry: "main.ts",
    },

    "tools/k6": {
      entry: "load.js",
    },
    "tools/migrate": {
      entry: "main.ts",
    },
  },
};

export default config;
