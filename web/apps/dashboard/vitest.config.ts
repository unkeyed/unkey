import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    environmentMatchGlobs: [
      ["**/*.test.tsx", "jsdom"],
      ["**/hooks/*.test.ts", "jsdom"],
      ["**/lib/collections/**/*.test.ts", "jsdom"],
      ["**/lib/identities-query.test.ts", "jsdom"],
      ["**/lib/portal/use-portal.test.ts", "jsdom"],
      ["**/lib/unkey-client.test.ts", "jsdom"],
    ],
    alias: { "@/": new URL("./", import.meta.url).pathname },
  },
});
