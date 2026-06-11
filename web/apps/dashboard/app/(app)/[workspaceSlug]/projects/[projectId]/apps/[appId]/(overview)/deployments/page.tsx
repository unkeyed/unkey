"use client";
import { Plus } from "@unkey/icons";
import {
  Button,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsCardList } from "./components/deployments-card-list";

export default function Deployments() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Deployments</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateDeploymentButton
            renderTrigger={({ onClick }) => (
              <Button size="md" variant="primary" onClick={onClick}>
                <Plus iconSize="sm-regular" />
                Create Deployment
              </Button>
            )}
          />
        </PageHeaderActions>
      </PageHeader>
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-5 px-6 pt-6 pb-20">
        <DeploymentsListControls />
        <DeploymentsCardList />
      </div>
    </PageContainer>
  );
}
