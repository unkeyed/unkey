import type { KnipConfig } from "knip";

const config: KnipConfig = {
  ignore: ["**/*.test.ts", "**/*.spec.ts", "packages/nuxt/**/*"],
  workspaces: {
    ".": {
      entry: "checkly.config.ts",
    },
    "apps/web": {
      entry: ["contentlayer.config.ts", "lib/trpc/index.ts", "trpc.config.ts"],
    },
    "apps/api": {
      entry: "src/worker.ts",
      ignore: ["**/*.test.ts", "src/pkg/testutil/*.ts"],
    },
    "internal/billing": {
      entry: "src/index.ts",
    },
    "internal/db": {
      entry: "src/index.ts",
    },
    "internal/hash": {
      entry: "src/index.ts",
    },
    "internal/id": {
      entry: "src/index.ts",
    },
    "internal/keys": {
      entry: "src/index.ts",
    },
    "internal/resend": {
      entry: "src/index.ts",
    },
    "internal/vercel": {
      entry: "src/index.ts",
    },
    "packages/nuxt": {
      entry: "src/module.ts",
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
