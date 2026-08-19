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

export const billingUIUpgrades = flag<boolean, Entities>({
  key: "billing-ui-upgrades",
  description: "Show the split Billing / Usage / Limits settings pages. Requires deploy-billing.",
  defaultValue: false,
  options: [
    { value: false, label: "Off" },
    { value: true, label: "On" },
  ],
  identify,
  adapter: adapter(),
});

export const showDarksoulsSuccessBanner = flag<boolean, Entities>({
  key: "show-darksouls-success-banner",
  description: "Show the animated deployment success easter egg.",
  defaultValue: false,
  options: [
    { value: false, label: "Off" },
    { value: true, label: "On" },
  ],
  identify,
  adapter: adapter(),
});

export const projectsNav = flag<boolean, Entities>({
  key: "projects-nav",
  description:
    "Use the projects-first navigation (sidebar, breadcrumbs, landing redirect). Off until rollout.",
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
