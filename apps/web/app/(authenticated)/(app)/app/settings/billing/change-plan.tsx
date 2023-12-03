"use client";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { type Workspace } from "@unkey/db";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
type Props = {
  trigger: React.ReactNode;
  workspace: Workspace;
};

export const ChangePlan: React.FC<Props> = ({ workspace, trigger }) => {
  const { toast } = useToast();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const changePlan = trpc.workspace.changePlan.useMutation({
    onSuccess: () => {
      toast({
        title: "Plan changed",
        description: "Your plan has been changed",
      });
      router.refresh();
      setOpen(false);
    },
    onError: (error) => {
      toast({
        title: "Error",
        description: error.message,
        variant: "alert",
      });
      setOpen(false);
    },
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Change plan</DialogTitle>
          <DialogDescription>
            You are currently on the <span className="font-bold capitalize">{workspace.plan}</span>{" "}
            plan.
          </DialogDescription>
          <DialogClose />
        </DialogHeader>

        <DialogFooter className="grid grid-cols-3 w-full gap-4">
          <Button
            className="col-span-1"
            variant={workspace.plan === "free" ? "disabled" : "secondary"}
            disabled={workspace.plan === "free"}
            onClick={() => changePlan.mutateAsync({ workspaceId: workspace.id, plan: "free" })}
          >
            {changePlan.isLoading ? <Loading /> : "Free"}
          </Button>
          <Button
            className="col-span-1"
            variant={workspace.plan === "pro" ? "disabled" : "secondary"}
            disabled={workspace.plan === "pro"}
            onClick={() => changePlan.mutateAsync({ workspaceId: workspace.id, plan: "pro" })}
          >
            {changePlan.isLoading ? <Loading /> : "Pro"}
          </Button>
          <Link href="mailto:support@unkey.dev">
            <Button
              className="col-span-1"
              variant={workspace.plan === "enterprise" ? "disabled" : "secondary"}
              disabled={workspace.plan === "enterprise"}
            >
              Enterprise
            </Button>
          </Link>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
