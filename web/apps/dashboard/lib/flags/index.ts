import { flag } from "flags/next";
import { type Entities, adapter, identify } from "./plumbing";

// Feature flag registry. To add a flag: declare it here with `flag<T, Entities>({...})`,
// then register it in ./resolve.ts so the FlagsProvider exposes it to client components.
// See docs/engineering/contributing/tooling/feature-flags.mdx for the full workflow.

export const helloWorld = flag<boolean, Entities>({
  key: "hello-world",
  description: "Smoke test for the flags pipeline",
  defaultValue: false,
  options: [
    { value: false, label: "Off" },
    { value: true, label: "On" },
  ],
  identify,
  adapter: adapter(),
});

// Gates whether extensions marked `mode: "live"` actually call the real
// install backend. On by default — turn off to force every extension into
// localStorage preview mode (useful for demos, screenshots, or to compare
// the wired-up flow against a stubbed one without backend writes).
export const extensionsLive = flag<boolean, Entities>({
  key: "extensions-live",
  description: "Enable real install backend for live extensions (e.g. log drains)",
  defaultValue: true,
  options: [
    { value: true, label: "Live backend" },
    { value: false, label: "Preview only" },
  ],
  identify,
  adapter: adapter(),
});
