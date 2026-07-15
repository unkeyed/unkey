"use client";
import { collection } from "@/lib/collections";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { Plus } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsCardList } from "./components/deployments-card-list";

export default function Deployments() {
  useCollectionPolling(() => collection.deployments.utils.refetch(), {
    intervalMs: 20_000,
    enabled: true,
  });

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
                Create deployment
              </Button>
            )}
          />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <DeploymentsListControls />
        <DeploymentsCardList />
      </PageBody>
    </PageContainer>
  );
}
