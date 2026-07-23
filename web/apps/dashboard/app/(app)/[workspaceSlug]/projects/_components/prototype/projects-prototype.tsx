"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { CreateProjectButton } from "../create-project-button";
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
import { PrototypeProvider, usePrototypeWorlds } from "./store";

export function ProjectsPrototype() {
  return (
    <PrototypeProvider>
      <ProjectsPrototypeInner />
    </PrototypeProvider>
  );
}

function ProjectsPrototypeInner() {
  const workspace = useWorkspaceNavigation();
  const { scenario, setScenario } = useScenario();
  const { variant, setVariant } = useRowVariant();
  const { mark, setMark } = useMark();
  const { agentStyle, setAgentStyle } = useAgentStyle();
  const [agentDismissed, setAgentDismissed] = useAgentDismissed();
  const { worlds, resetWorlds } = usePrototypeWorlds();

  const world = worlds[scenario];

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
            {world.projects.length === 0 ? (
              <ProjectsEmptyCard>
                <CreateProjectButton workspaceSlug={workspace.slug} variant="outline" />
              </ProjectsEmptyCard>
            ) : (
              <ProjectGrid projects={world.projects} />
            )}
          </div>
          <Rail
            data={world}
            variant={variant}
            mark={mark}
            agentStyle={agentStyle}
            workspaceSlug={workspace.slug}
            agentDismissed={agentDismissed}
            onDismissAgent={() => setAgentDismissed(true)}
          />
        </div>
      </PageBody>
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
        onReset={resetWorlds}
      />
    </PageContainer>
  );
}
