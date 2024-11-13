import type { KnipConfig } from "knip";

const config: KnipConfig = {
  ignore: [
    "tools/artillery/**", // workspace but no package.json
  ],
  ignoreWorkspaces: [
    "apps/agent", // golang app
  ],
  workspaces: {
    ".": {
      entry: "checkly.config.ts",
    },
    "apps/billing": {
      entry: [
        // Knip doesn't have a trigger.dev plugin, so I'm guessing the entry points here:
        "trigger.config.ts",
        "src/trigger/*.ts",
      ],
    },
    "apps/dashboard": {
      entry: [
        // Knip doesn't have a @trpc/* plugin, so I'm guessing the entry points here:
        "app/**/*.{ts,tsx}",
        "lib/trpc/{client,server}.ts",
        "trpc.config.ts",
      ],
    },
    "tools/k6": {
      entry: "load.js",
    },
  },
};

export default config;
