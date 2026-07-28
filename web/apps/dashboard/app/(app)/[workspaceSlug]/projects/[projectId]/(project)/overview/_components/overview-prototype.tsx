"use client";

import {
  useChartScheme,
  useOverviewLayout,
  useScenario,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";
import {
  PrototypeProvider,
  usePrototypeWorlds,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";
import { BookBookmark } from "@unkey/icons";
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

const DOCS_URL = "https://unkey.com/docs";

export function OverviewPrototype() {
  return (
    <PrototypeProvider>
      <OverviewPrototypeInner />
    </PrototypeProvider>
  );
}

function OverviewPrototypeInner() {
  const { scenario, setScenario } = useScenario();
  const { layout, setLayout } = useOverviewLayout();
  const { chartScheme, setChartScheme } = useChartScheme();
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
            variant="outline"
            render={<Link href={DOCS_URL} target="_blank" rel="noopener noreferrer" />}
          >
            <BookBookmark iconSize="sm-regular" />
            Documentation
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <ProjectOverview data={data} layout={layout} />
      </PageBody>
      <OverviewDebugCommand
        scenario={scenario}
        onScenario={setScenario}
        layout={layout}
        onLayout={setLayout}
        chartScheme={chartScheme}
        onChartScheme={setChartScheme}
        onReset={resetWorlds}
      />
    </PageContainer>
  );
}
