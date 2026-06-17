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

export const rootKeyUrnPermissions = flag<boolean, Entities>({
  key: "root-key-urn-permissions",
  description: "Enable the URN root key permission composer",
  defaultValue: false,
  options: [
    { value: false, label: "Legacy permissions" },
    { value: true, label: "URN permissions" },
  ],
  identify,
  adapter: adapter(),
});
