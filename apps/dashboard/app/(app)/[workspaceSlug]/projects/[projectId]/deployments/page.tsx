"use client";
import { cn } from "@unkey/ui/src/lib/utils";
import { useProjectLayout } from "../layout-provider";
import { DeploymentsListControlCloud } from "./components/control-cloud";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsList } from "./components/table/deployments-list";

export default function Deployments() {
  const { isDetailsOpen } = useProjectLayout();

  return (
    <div
      className={cn(
        "flex flex-col transition-all duration-300 ease-in-out",
        isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]",
      )}
    >
      <DeploymentsListControls />
      <DeploymentsListControlCloud />
      <DeploymentsList />
    </div>
  );
}
