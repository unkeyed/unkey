import type { PropsWithChildren } from "react";
import { DeploymentLayoutProvider } from "./layout-provider";

export default function DeploymentLayout({ children }: PropsWithChildren) {
  // The breadcrumb + tabs render in the parent app layout, above the scroll
  // container, so they stay fixed. This layout just provides deployment data.
  return <DeploymentLayoutProvider>{children}</DeploymentLayoutProvider>;
}
