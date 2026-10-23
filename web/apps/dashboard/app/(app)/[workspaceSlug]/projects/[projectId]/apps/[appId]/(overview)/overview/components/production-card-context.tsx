"use client";

import type { AppSource } from "@/components/deploy/app-source";
import type { Deployment, DeploymentSummary } from "@/lib/collections";
import type { Route } from "next";
import { createContext, use } from "react";
import type { DisplayDomain } from "../../../components/display-domain";
import type { ProductionPolicies } from "../../policies/hooks/use-policies-data";
import type { DeploymentDisplayStatus } from "./status";

export type CanvasModel = {
  appName: string;
  appSource: AppSource;
  defaultBranch: string | null;
  domains: ReadonlyArray<DisplayDomain>;
  domainsHref: Route;
  policiesHref: Route;
  policies: ProductionPolicies;
};

export type ProductionCardContextValue = {
  deployment: Deployment;
  status: DeploymentDisplayStatus;
  isCurrent: boolean;
  isRolledBack: boolean;
  canvas: CanvasModel;
  sourceRepo: string | null;
  diagnostic: { label: string; href: Route } | null;
  deploymentHref: Route;
  logsHref: Route;
  requestsHref: Route;
  rollbackTarget: DeploymentSummary | undefined;
  undoCandidates: DeploymentSummary[];
  openRollback: () => void;
  openUndo: () => void;
};

const ProductionCardContext = createContext<ProductionCardContextValue | null>(null);

export const ProductionCardProvider = ProductionCardContext.Provider;

export function useProductionCard(): ProductionCardContextValue {
  const ctx = use(ProductionCardContext);
  if (!ctx) {
    throw new Error("useProductionCard must be used within AppProductionCard");
  }
  return ctx;
}
