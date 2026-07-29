"use client";

import { useScenario } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";
import {
  PrototypeProvider,
  usePrototypeWorlds,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";
import { routes } from "@/lib/navigation/routes";
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
import Link from "next/link";
import { useOverviewProjectData } from "./overview-data";
import { OverviewDebugCommand } from "./overview-debug-command";
import { ProjectOverview } from "./project-overview";

export function OverviewPrototype() {
  return (
    <PrototypeProvider>
      <OverviewPrototypeInner />
    </PrototypeProvider>
  );
}

function OverviewPrototypeInner() {
  const { scenario, setScenario } = useScenario();
  const { resetWorlds } = usePrototypeWorlds();
  const data = useOverviewProjectData();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>{data.project.name}</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            size="md"
            render={
              <Link
                href={routes.projects.apps.new({
                  workspaceSlug: data.workspaceSlug,
                  projectId: data.project.id,
                })}
              />
            }
          >
            <Plus iconSize="sm-regular" />
            Create app
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <ProjectOverview data={data} />
      </PageBody>
      <OverviewDebugCommand scenario={scenario} onScenario={setScenario} onReset={resetWorlds} />
    </PageContainer>
  );
}
