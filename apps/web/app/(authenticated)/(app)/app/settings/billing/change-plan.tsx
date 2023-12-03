"use client";

import { Confirm } from "@/components/dashboard/confirm";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { type Workspace } from "@unkey/db";
import { useRouter } from "next/navigation";

import React from "react";
type Props = {
  label: string;
  workspace: Workspace;
  plan: Workspace["plan"];
};

export const ChangePlan: React.FC<Props> = ({ workspace, plan, label }) => {
  const { toast } = useToast();
  const router = useRouter();
  const changePlan = trpc.workspace.changePlan.useMutation({
    onSuccess: () => {
      toast({
        title: "Plan changed",
        description: "Your plan has been changed",
      });
      router.refresh();
    },
    onError: (error) => {
      toast({
        title: "Error",
        description: error.message,
        variant: "alert",
      });
    },
  });
  return (
    <Confirm
      title={`Change plan to ${plan}`}
      description="Are you sure you want to change your plan?"
      disabled={workspace.plan === plan}
      onConfirm={() => changePlan.mutateAsync({ workspaceId: workspace.id, plan })}
      trigger={<Button type="button">{label}</Button>}
    />
  );
};
