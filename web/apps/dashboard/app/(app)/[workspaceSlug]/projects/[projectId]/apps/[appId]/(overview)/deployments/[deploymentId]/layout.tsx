import { PageContainer } from "@unkey/ui";
import type { PropsWithChildren } from "react";
import { DeploymentDetailHeader } from "./deployment-detail-header";
import { DeploymentLayoutProvider } from "./layout-provider";

export default function DeploymentLayout({ children }: PropsWithChildren) {
  return (
    <DeploymentLayoutProvider>
      <PageContainer>
        <DeploymentDetailHeader />
        {children}
      </PageContainer>
    </DeploymentLayoutProvider>
  );
}
