import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["main.ts"],
  format: ["esm"],
  splitting: false,
  sourcemap: true,
  clean: true,
  bundle: true,
  dts: true,
});
