import type { PropsWithChildren } from "react";
import { DeploymentLayoutProvider } from "./layout-provider";

export default function DeploymentLayout({ children }: PropsWithChildren) {
  return (
    <DeploymentLayoutProvider>
      <div className="flex flex-col h-full">
        <div id="deployment-scroll-container" className="flex-1 overflow-auto">
          {children}
        </div>
      </div>
    </DeploymentLayoutProvider>
  );
}
