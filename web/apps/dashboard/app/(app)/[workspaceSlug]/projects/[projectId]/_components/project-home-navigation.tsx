"use client";
import { DeployFeedbackButton } from "@/components/dashboard/deploy-feedback-button";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Cube, Plus } from "@unkey/icons";

type ProjectHomeNavigationProps = {
  projectId: string;
  onCreateApp: () => void;
};

export const ProjectHomeNavigation = ({ projectId, onCreateApp }: ProjectHomeNavigationProps) => {
  const workspace = useWorkspaceNavigation();
  const basePath = `/${workspace.slug}/projects`;

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube iconSize="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link href={basePath} noop={false} active={false} isLast={false}>
          Projects
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`${basePath}/${projectId}`} noop active isLast>
          Apps
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      {/* <Navbar.Actions> */}
      {/*   <NavbarActionButton title="Create new app" onClick={onCreateApp}> */}
      {/*     <Plus /> */}
      {/*     Create app */}
      {/*   </NavbarActionButton> */}
      {/*   <DeployFeedbackButton /> */}
      {/* </Navbar.Actions> */}
    </Navbar>
  );
};
