"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { CreateProjectButton } from "../create-project-button";
import { LiveRail } from "./live-rail";
import { MOCK, type ProjectMock } from "./mock-data";
import { ProjectGrid, ProjectsEmptyCard } from "./project-grid";
import { Rail } from "./rail";
import {
  DebugCommand,
  useAgentDismissed,
  useAgentStyle,
  useMark,
  useRowVariant,
  useScenario,
} from "./scenario";

export function ProjectsPrototype() {
  const workspace = useWorkspaceNavigation();
  const { scenario, setScenario } = useScenario();
  const { variant, setVariant } = useRowVariant();
  const { mark, setMark } = useMark();
  const { agentStyle, setAgentStyle } = useAgentStyle();
  const [agentDismissed, setAgentDismissed] = useAgentDismissed();

  const projectsQuery = useLiveQuery((q) => q.from({ project: collection.projects }));
  const mockData = scenario === "live" ? null : MOCK[scenario];

  const liveProjects: ProjectMock[] = (projectsQuery.data ?? []).map((p) => ({
    id: p.id,
    name: p.name,
    subtitle: p.appCount > 0 ? `${p.appCount} app${p.appCount === 1 ? "" : "s"}` : "No apps yet",
    apps: (p.apps ?? []).map((a) => ({ id: a.id, source: a.source })),
    appCount: p.appCount,
  }));

  const gridProjects = mockData ? mockData.projects : liveProjects;

  const debugBar = (
    <DebugCommand
      scenario={scenario}
      onScenario={setScenario}
      variant={variant}
      onVariant={setVariant}
      mark={mark}
      onMark={setMark}
      agentStyle={agentStyle}
      onAgentStyle={setAgentStyle}
      agentDismissed={agentDismissed}
      onToggleAgent={() => setAgentDismissed(!agentDismissed)}
    />
  );

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Projects</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateProjectButton workspaceSlug={workspace.slug} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col-reverse lg:flex-row items-start gap-6">
          <div className="flex-1 min-w-0 w-full">
            {gridProjects.length === 0 ? (
              <ProjectsEmptyCard>
                <CreateProjectButton workspaceSlug={workspace.slug} variant="outline" />
              </ProjectsEmptyCard>
            ) : (
              <ProjectGrid projects={gridProjects} />
            )}
          </div>
          {mockData ? (
            <Rail
              data={mockData}
              variant={variant}
              mark={mark}
              agentStyle={agentStyle}
              workspaceSlug={workspace.slug}
              agentDismissed={agentDismissed}
              onDismissAgent={() => setAgentDismissed(true)}
            />
          ) : (
            <LiveRail
              variant={variant}
              mark={mark}
              agentStyle={agentStyle}
              workspaceSlug={workspace.slug}
              agentDismissed={agentDismissed}
              onDismissAgent={() => setAgentDismissed(true)}
            />
          )}
        </div>
      </PageBody>
      {debugBar}
    </PageContainer>
  );
}
