import { firewallActionSchema } from "@/lib/collections/deploy/policies.schema";
import { IconBanOutline18 } from "@unkey/icons";
import { z } from "zod";
import { sharedFormFields } from "../shared";
import { type PolicyKindModel, statusLabel } from "../types";

// Firewall has a single action today (DENY) and no other configuration.
// The action is kept on the form so the wire payload stays self-describing
// and so adding more actions later is purely additive.
export const firewallFormSchema = z.object({
  ...sharedFormFields,
  type: z.literal("firewall"),
  action: firewallActionSchema,
});

type FirewallFormValues = z.infer<typeof firewallFormSchema>;

export const firewall: PolicyKindModel<"firewall", FirewallFormValues> = {
  label: "Firewall",
  Icon: IconBanOutline18,
  does: "Blocks every request that matches its conditions.",
  rejects: [{ status: 403, reason: "Every request that matches" }],
  defaults: () => ({ type: "firewall", action: "ACTION_DENY" }),
  toWire: (v) => ({ type: "firewall", firewall: { action: v.action } }),
  fromWire: (p) => ({ type: "firewall", action: p.firewall.action }),
  summary: () => `Deny with ${statusLabel(403)}`,
};
