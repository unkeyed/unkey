import type { EnvironmentKind } from "@/lib/collections/deploy/environments";

export function followsProduction(targetName: string): string {
  return `Follows ${targetName}’s live production deployment, including rollbacks.`;
}

type CallerKind = {
  label: string;
  fieldLabel: string;
  automatic: {
    label: string;
    description: (targetName: string) => string;
    explainInHint: boolean;
  };
  coveredByAutomatic: EnvironmentKind | null;
  needsRedeploy: boolean;
};

export const CALLER_KINDS: Record<EnvironmentKind, CallerKind> = {
  production: {
    label: "Production",
    fieldLabel: "Production target",
    automatic: { label: "Production", description: followsProduction, explainInHint: false },
    coveredByAutomatic: "production",
    needsRedeploy: true,
  },
  preview: {
    label: "Preview",
    fieldLabel: "Preview target",
    automatic: {
      label: "Same Git branch",
      description: (targetName) =>
        `Each preview connects to the newest ${targetName} preview on the same Git branch. If there is none, it has no target.`,
      explainInHint: true,
    },
    coveredByAutomatic: null,
    needsRedeploy: false,
  },
};
