import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import { nitroV2Plugin } from "@tanstack/nitro-v2-vite-plugin";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  server: {
    allowedHosts: process.env.AMP_ORB ? [".e2b.app", ".onamp.dev"] : undefined,
    port: 3100,
  },
  resolve: {
    alias: {
      "~": path.resolve(import.meta.dirname, "./src"),
    },
  },
  plugins: [
    tailwindcss(),
    tanstackStart(),
    // The Nitro server bundle otherwise inherits Vite's browser build target,
    // which esbuild can't lower; pin it to the Node runtime instead.
    nitroV2Plugin({ preset: "node-server", esbuild: { options: { target: "node22" } } }),
    viteReact(),
  ],
});
