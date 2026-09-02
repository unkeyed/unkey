"use client";
import { collection } from "@/lib/collections";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  ResourceList,
} from "@unkey/ui";
import { IconPlusOutline18 } from "nucleo-ui-outline-18";
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
                <IconPlusOutline18 />
                Create deployment
              </Button>
            )}
          />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <ResourceList>
          <DeploymentsListControls />
          <DeploymentsCardList />
        </ResourceList>
      </PageBody>
    </PageContainer>
  );
}
