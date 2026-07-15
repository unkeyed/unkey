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

// deployBilling gates the entire Unkey Deploy billing UI (subscribe / change /
// cancel, usage, credits). Defaults off so prod shows nothing until we flip
// workspaces in for the GA rollout.
export const deployBilling = flag<boolean, Entities>({
  key: "deploy-billing",
  description: "Show the Unkey Deploy billing UI. Off until GA rollout.",
  defaultValue: false,
  options: [
    { value: false, label: "Off" },
    { value: true, label: "On" },
  ],
  identify,
  adapter: adapter(),
});

export const appOverview = flag<boolean, Entities>({
  key: "app-overview",
  description:
    "Show the app overview page and use it as the default app landing. Off until rollout.",
  defaultValue: false,
  options: [
    { value: false, label: "Off" },
    { value: true, label: "On" },
  ],
  identify,
  adapter: adapter(),
});

// portalManagement gates the portal configuration page and its sidebar nav
// item. Off until portal GA so it can be developed and merged without being
// visible. Enable per-workspace to roll out to internal workspaces first.
export const portalManagement = flag<boolean, Entities>({
  key: "portal-management",
  description: "Show the portal configuration page in the dashboard sidebar. Off until portal GA.",
  defaultValue: false,
  options: [
    { value: false, label: "Off" },
    { value: true, label: "On" },
  ],
  identify,
  adapter: adapter(),
});
