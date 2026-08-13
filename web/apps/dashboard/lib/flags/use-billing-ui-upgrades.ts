"use client";

import { useFlag } from "@/lib/flags/provider";

export function useBillingUIUpgrades(): boolean {
  const deployBilling = useFlag("deployBilling");
  const billingUIUpgrades = useFlag("billingUIUpgrades");
  return deployBilling && billingUIUpgrades;
}
