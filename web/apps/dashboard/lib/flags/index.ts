import { flag } from "flags/next";
import { type Entities, adapter, identify } from "./plumbing";

// How to add a dashboard feature flag.
//
// This file is the registry for flags that application code can import from
// `@/lib/flags`. Keep generic setup in ./plumbing.ts, and keep flag declarations
// here so the discovery endpoint exports only actual flags. See
// docs/engineering/contributing/tooling/feature-flags.mdx for the full workflow.
//
// 1. Create the flag in Vercel from `web/apps/dashboard`:
//
//    vercel flags create my-flag --kind boolean --description "What it gates"
//
// 2. Declare the flag in this file. Use the exact Vercel key, add a clear
//    description, set a safe defaultValue, include all known options, pass the
//    shared identify helper, and use the shared adapter wrapper.
//
//    export const myFlag = flag<boolean, Entities>({
//      key: "my-flag",
//      description: "Gates the new dashboard workflow",
//      defaultValue: false,
//      options: [
//        { value: false, label: "Off" },
//        { value: true, label: "On" },
//      ],
//      identify,
//      adapter: adapter(),
//    });
//
// 3. Choose defaultValue as the safe self-hosted and local-development value.
//    When FLAGS is missing, the adapter wrapper uses a noop adapter that returns
//    defaultValue and does not load Vercel targeting rules.
//
// 4. Use the exported flag only from server-side code:
//
//    import { myFlag } from "@/lib/flags";
//
//    const enabled = await myFlag();
//
// 5. If the flag needs user or org targeting, configure rules in Vercel against
//    user.id or org.id. Add fields to Entities only when the provider needs more
//    stable targeting inputs.

// helloWorld verifies the dashboard can evaluate boolean feature flags through
// Vercel Flags while falling back to the declared default when setup is absent.
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
