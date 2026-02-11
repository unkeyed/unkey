import { DeploymentNavbar } from "./(overview)/navigations/deployment-navbar";
import { DeploymentLayoutProvider } from "./layout-provider";

export default function DeploymentLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <DeploymentLayoutProvider>
      <div className="flex flex-col h-full">
        <DeploymentNavbar />
        <div id="deployment-scroll-container" className="flex-1 overflow-auto">
          {children}
        </div>
      </div>
    </DeploymentLayoutProvider>
  );
}
