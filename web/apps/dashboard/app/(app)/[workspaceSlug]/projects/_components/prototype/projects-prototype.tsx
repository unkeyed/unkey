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
import { useState } from "react";
import { CreateProjectButton } from "../create-project-button";
import type { AgentStyle } from "./agent-setup";
import type { Mark } from "./marks";
import { ProjectGrid, ProjectsEmptyCard } from "./project-grid";
import { Rail } from "./rail";
import type { RowVariant } from "./scenario";
import { DebugCommand, useScenario } from "./scenario";
import { PrototypeProvider, usePrototypeWorlds } from "./store";

// Settled treatments: divided list rows, bar charts, the compact agent strip.
const ROW_VARIANT: RowVariant = "list";
const ROW_MARK: Mark = "bars";
const AGENT_STYLE: AgentStyle = "minimal";

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
  // Dismissal is the card's own behaviour, not a shared setting, so it stops at
  // component state — no URL param, no localStorage.
  const [agentDismissed, setAgentDismissed] = useState(false);
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
              <ProjectGrid projects={world.projects} workspaceSlug={workspace.slug} />
            )}
          </div>
          <Rail
            data={world}
            variant={ROW_VARIANT}
            mark={ROW_MARK}
            agentStyle={AGENT_STYLE}
            workspaceSlug={workspace.slug}
            agentDismissed={agentDismissed}
            onDismissAgent={() => setAgentDismissed(true)}
          />
        </div>
      </PageBody>
      <DebugCommand scenario={scenario} onScenario={setScenario} onReset={resetWorlds} />
    </PageContainer>
  );
}
