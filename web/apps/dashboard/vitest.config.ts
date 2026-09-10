import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "jsdom",
    alias: {
      "@/": new URL("./", import.meta.url).pathname,
      react: new URL("./node_modules/react", import.meta.url).pathname,
      "react-dom": new URL("./node_modules/react-dom", import.meta.url).pathname,
      "@base-ui/react": new URL("./node_modules/@base-ui/react", import.meta.url).pathname,
    },
  },
});
