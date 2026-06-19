"use client";

import type { Deployment } from "@/lib/collections";
import type { Route } from "next";
import { createContext, use } from "react";
import type { Pulse } from "./g-pulse";
import type { DeploymentDisplayStatus } from "./status";

export type CardDomain = { hostname: string; url: string };

export type ProductionCardContextValue = {
  deployment: Deployment;
  status: DeploymentDisplayStatus;
  isRolledBack: boolean;
  rolledBackFrom: { commitSha: string | null; commitMessage: string | null } | null;
  sourceRepo: string | null;
  primaryDomain: CardDomain | null;
  additionalDomains: CardDomain[];
  addCustomDomainHref: string | null;
  diagnostic: { label: string; href: Route } | null;
  logsHref: Route;
  requestsHref: Route;
  rollbackTarget: Deployment | undefined;
  undoCandidates: Deployment[];
  pulse: Pulse;
  isChartLoading: boolean;
  isChartError: boolean;
  openRollback: () => void;
  openUndo: () => void;
};

const ProductionCardContext = createContext<ProductionCardContextValue | null>(null);

export const ProductionCardProvider = ProductionCardContext.Provider;

export function useProductionCard(): ProductionCardContextValue {
  const ctx = use(ProductionCardContext);
  if (!ctx) {
    throw new Error("useProductionCard must be used within ProductionDeploymentCard");
  }
  return ctx;
}
