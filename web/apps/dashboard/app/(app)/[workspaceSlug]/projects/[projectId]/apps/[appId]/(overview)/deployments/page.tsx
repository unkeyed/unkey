"use client";
import { useFlag } from "@/lib/flags/provider";
import { IconPlusOutline18 } from "@unkey/icons";
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
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { DeploymentsListControls } from "./components/controls";
import { DeploymentsCardList } from "./components/deployments-card-list";
import { DeploymentsCardList as LegacyDeploymentsCardList } from "./components/legacy/deployments-card-list";

export default function Deployments() {
  const appV2 = useFlag("appV2");
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Deployments</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateDeploymentButton
            renderTrigger={({ onClick }) => (
              <Button size="sm" variant="primary" onClick={onClick}>
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
          {appV2 ? <DeploymentsCardList /> : <LegacyDeploymentsCardList />}
        </ResourceList>
      </PageBody>
    </PageContainer>
  );
}
