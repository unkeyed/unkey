// @vitest-environment node

import { createRequire } from "node:module";
import { PHASE_DEVELOPMENT_SERVER, PHASE_PRODUCTION_BUILD } from "next/constants";
import loadConfig from "next/dist/server/config";
import { expect, it } from "vitest";

const { webpack } = createRequire(import.meta.url)("next/dist/compiled/webpack/webpack");

it.each([
  { name: "development", dev: true, phase: PHASE_DEVELOPMENT_SERVER },
  { name: "production", dev: false, phase: PHASE_PRODUCTION_BUILD },
] as const)("uses the $name cache policy through the Sentry wrapper", async ({ dev, phase }) => {
  const dir = new URL("./", import.meta.url).pathname;
  const config = await loadConfig(phase, dir);
  const cache = {
    type: "filesystem",
    maxMemoryGenerations: 0,
    cacheDirectory: "/tmp/unkey-webpack-cache-test",
  };

  const result = config.webpack?.(
    { cache, module: { rules: [] }, plugins: [] },
    {
      dir,
      dev,
      isServer: true,
      buildId: "cache-test",
      config,
      defaultLoaders: { babel: {} },
      totalPages: 0,
      webpack,
      nextRuntime: "nodejs",
    },
  );

  if (dev) {
    expect(result?.cache).toBe(cache);
  } else {
    expect(result?.cache).toEqual({ type: "memory" });
  }
});
