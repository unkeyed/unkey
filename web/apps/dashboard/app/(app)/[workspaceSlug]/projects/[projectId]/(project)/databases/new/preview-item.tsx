"use client";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
} from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { type EnvironmentPlan, type SourceChoice, sourceFlow } from "./create-flow-model";
import { EnvironmentHeading } from "./fields";

export function PreviewItem({
  source,
  plan,
  productionBranch,
  detail,
  trailing,
  onPlan,
  children,
}: {
  source: SourceChoice;
  plan: EnvironmentPlan;
  productionBranch: string;
  detail?: ReactNode;
  trailing?: ReactNode;
  onPlan: (plan: EnvironmentPlan) => void;
  children?: ReactNode;
}) {
  const [confirming, setConfirming] = useState(false);
  const rows: Record<
    EnvironmentPlan,
    { badge: ReactNode; detail: ReactNode; trailing: ReactNode; body: ReactNode }
  > = {
    production_preview: {
      badge: null,
      detail,
      trailing,
      body: children ? (
        <>
          {children}
          <Button
            variant="ghost"
            color="danger"
            size="sm"
            className="-ml-2 self-start"
            onClick={() => setConfirming(true)}
          >
            Remove preview branch
          </Button>
        </>
      ) : null,
    },
    production: {
      badge: <Badge size="sm">Deactivated</Badge>,
      detail: `Preview deployments use ${productionBranch}.`,
      trailing: (
        <Button variant="outline" size="sm" onClick={() => onPlan("production_preview")}>
          Add preview branch
        </Button>
      ),
      body: null,
    },
  };
  const content = rows[plan];
  return (
    <>
      <div
        data-plan={plan}
        className="flex w-full flex-col gap-4 rounded-lg border px-4 py-3 data-[plan=production]:border-dashed data-[plan=production]:bg-grayA-2"
      >
        <div className="flex items-center gap-2 text-sm">
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="flex items-center gap-2">
              <EnvironmentHeading env="preview" />
              {content.badge}
            </span>
            {content.detail ? (
              <span className="pl-6 text-xs text-gray-11">{content.detail}</span>
            ) : null}
          </div>
          {content.trailing}
        </div>
        {content.body}
      </div>
      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Use production data for previews?</AlertDialogTitle>
            <AlertDialogDescription>
              Preview deployments will read and write the {productionBranch} branch. A bad migration
              in a pull request changes production.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction color="danger" onClick={() => onPlan("production")}>
              {sourceFlow[source].previewConfirm}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
