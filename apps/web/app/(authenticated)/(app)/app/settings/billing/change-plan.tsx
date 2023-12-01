"use client";

import { Confirm } from "@/components/dashboard/confirm";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { type Workspace } from "@unkey/db";
import { BatteryFull, BatteryLow, BatteryMedium, LucideIcon } from "lucide-react";
import { useRouter } from "next/navigation";

import React from "react";
type Props = {
  workspace: Workspace;
};

export const ChangePlan: React.FC<Props> = ({ workspace }) => {
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
    <Card>
      <CardHeader>
        <CardTitle>Plan</CardTitle>
        <CardDescription>
          You are currently on the <strong className="capitalize">{workspace.plan} </strong> plan.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-8 max-sm:gap-2 ">
          <Confirm
            title="Change plan to free"
            description="Are you sure you want to change your plan?"
            disabled={workspace.plan === "free"}
            onConfirm={() => changePlan.mutateAsync({ workspaceId: workspace.id, plan: "free" })}
            trigger={<Option currentPlan={workspace.plan} plan="free" icon={BatteryLow} />}
          />
          <Confirm
            title="Change plan to pro"
            description="Are you sure you want to change your plan?"
            disabled={workspace.plan === "pro"}
            onConfirm={() => changePlan.mutateAsync({ workspaceId: workspace.id, plan: "pro" })}
            trigger={<Option currentPlan={workspace.plan} plan="pro" icon={BatteryMedium} />}
          />

          <a href="mailto:support@unkey.dev">
            <Option currentPlan={workspace.plan} plan="enterprise" icon={BatteryFull} />
          </a>
        </div>
      </CardContent>
    </Card>
  );
};

type OptionProps = {
  currentPlan: Workspace["plan"];
  plan: Workspace["plan"];
  icon: LucideIcon;
};

const Option: React.FC<OptionProps> = (props) => {
  return (
    <div
      className={cn(
        "border text-sm rounded-md hover:border-primary flex items-center justify-center gap-2 h-8 p-2 ",
        {
          "bg-primary text-primary-foreground border-primary": props.plan === props.currentPlan,
        },
      )}
    >
      <props.icon className="w-4 h-4 shrink-0" /> <span className="capitalize">{props.plan}</span>
    </div>
  );
};
