"use client";
import type { PropsWithChildren } from "react";
import { RatelimitOverviewLogsControlCloud } from "../[namespaceId]/_overview/components/control-cloud";
import { RatelimitOverviewLogsControls } from "../[namespaceId]/_overview/components/controls";

export const RatelimitClient = ({ children }: PropsWithChildren) => {
  return (
    <div className="flex flex-col">
      <RatelimitOverviewLogsControls />
      <RatelimitOverviewLogsControlCloud />
      <div className="p-5">{children}</div>
    </div>
  );
};
