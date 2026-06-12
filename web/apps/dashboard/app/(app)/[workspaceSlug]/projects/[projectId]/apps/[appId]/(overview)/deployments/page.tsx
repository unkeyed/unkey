"use client";
import { Plus } from "@unkey/icons";
import {
  Button,
  PageBody,
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
          <PageHeaderDescription>View and manage deployments for this app.</PageHeaderDescription>
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
      <PageBody className="flex flex-col gap-5 pt-6 pb-20">
        <DeploymentsListControls />
        <DeploymentsCardList />
      </PageBody>
    </PageContainer>
  );
}
