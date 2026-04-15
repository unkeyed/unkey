"use client";

import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { CreateDeploymentButton } from "../../navigations/create-deployment-button";

export function DeploymentsHeader() {
  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div className="flex flex-col gap-0.5">
        <h1 className="font-semibold text-gray-12 text-lg leading-8">Deployments</h1>
        <p className="text-[13px] text-gray-11 leading-5">
          View and manage deployments for this project.
        </p>
      </div>
      <CreateDeploymentButton
        renderTrigger={({ onClick }) => (
          <Button size="md" variant="primary" onClick={onClick}>
            <Plus iconSize="sm-regular" />
            Create Deployment
          </Button>
        )}
      />
    </div>
  );
}
