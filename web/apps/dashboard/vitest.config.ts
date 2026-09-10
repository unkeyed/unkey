import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: { dedupe: ["react", "react-dom"] },
  test: {
    server: { deps: { inline: [/@base-ui\//] } },
    environment: "jsdom",
    alias: { "@/": new URL("./", import.meta.url).pathname },
  },
});
