"use client";

import { Loading } from "@/components/dashboard/loading";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTrigger,
} from "@/components/ui/dialog";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { PostHogEvent } from "@/providers/PostHogProvider";
import type { Workspace } from "@unkey/db";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
type Props = {
  newPlan: "free" | "pro";
  workspace: Workspace;
  label: string;
};

export const ChangePlanButton: React.FC<Props> = ({ workspace, newPlan, label }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const changePlan = trpc.workspace.updatePlan.useMutation({
    onSuccess: (data, variables, _context) => {
      toast.success(data.title, {
        description: data.message,
      });
      PostHogEvent({
        name: "plan_changed",
        properties: { plan: variables.plan, workspace: variables.workspaceId },
      });
      setOpen(false);
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const handleClick = () => {
    const hasPaymentMethod = !!workspace.stripeCustomerId;
    if (!hasPaymentMethod && newPlan === "pro") {
      return router.push(`/settings/billing/stripe?new_plan=${newPlan}`);
    }

    changePlan.mutateAsync({
      workspaceId: workspace.id,
      plan: newPlan === "free" ? "free" : "pro",
    });
  };

  const isSamePlan = workspace.plan === newPlan;
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild disabled={isSamePlan && !workspace.planDowngradeRequest}>
        <Button
          className="w-full"
          disabled={isSamePlan}
          variant={
            workspace.planDowngradeRequest ? "primary" : newPlan === "pro" ? "primary" : "default"
          }
        >
          {workspace.planDowngradeRequest ? "Resubscribe" : isSamePlan ? "Current Plan" : label}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          Do you want to {workspace.planDowngradeRequest ? "resubscribe" : "switch"} to the{" "}
          {newPlan} plan?
        </DialogHeader>
        {workspace.planDowngradeRequest ? null : (
          <Alert>
            <AlertTitle>Warning</AlertTitle>
            <AlertDescription>
              {newPlan === "free"
                ? "Your workspace will downgraded at the end of the month, you have access to all features of your current plan until then"
                : newPlan === "pro"
                  ? `You are about to switch to the ${newPlan} plan. Please note that you can not change your plan in the current cycle, contact support@unkey.dev if you need to.`
                  : ""}
            </AlertDescription>
          </Alert>
        )}

        <DialogFooter className="justify-end">
          <Button className="col-span-1" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button className="col-span-1" variant="primary" onClick={handleClick}>
            {changePlan.isLoading ? <Loading /> : "Switch"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
