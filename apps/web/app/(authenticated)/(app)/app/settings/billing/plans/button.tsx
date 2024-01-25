"use client";

import { Loading } from "@/components/dashboard/loading";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
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
import { type Workspace } from "@unkey/db";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
type Props = {
  newPlan: "free" | "pro";
  workspace: Workspace;
  label: string;
};

export const ChangePlanButton: React.FC<Props> = ({ workspace, newPlan, label }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const changePlan = trpc.workspace.changePlan.useMutation({
    onSuccess: (_data, variables, _context) => {
      toast.success("Your plan has been changed");
      PostHogEvent({
        name: "plan_changed",
        properties: { plan: variables.plan, workspace: variables.workspaceId },
      });
      router.refresh();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger>
        <Button
          className="w-full"
          variant={
            workspace.plan === newPlan ? "disabled" : newPlan === "pro" ? "primary" : "secondary"
          }
          disabled={workspace.plan === newPlan}
        >
          {label}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>Do you want to switch to the {newPlan} plan?</DialogHeader>
        <Alert>
          <AlertTitle>Warning</AlertTitle>
          <AlertDescription>
            You are about to switch to our {newPlan} plan. Please note there is a 24 hour pause
            before you can switch plans again.
          </AlertDescription>
        </Alert>

        <DialogFooter className="justify-end">
          <Button className="col-span-1" variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            className="col-span-1"
            variant="primary"
            disabled={workspace.plan === newPlan}
            onClick={() =>
              changePlan.mutateAsync({
                workspaceId: workspace.id,
                plan: newPlan === "free" ? "free" : "pro",
              })
            }
          >
            {changePlan.isLoading ? <Loading /> : "Switch"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
