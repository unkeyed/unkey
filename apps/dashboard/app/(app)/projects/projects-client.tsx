"use client";

import { LogsProvider } from "../logs/context/logs";
import { LogsControlCloud } from "./_components/control-cloud";
import { LogsControls } from "./_components/controls";
import { ProjectsList } from "./_components/list";
import { ProjectsNavigation } from "./navigation";

export function ProjectsClient() {
  return (
    <div>
      <ProjectsNavigation />
      <LogsProvider>
        <LogsControls />
        <LogsControlCloud />
      </LogsProvider>
      {/*Container*/}
      <ProjectsList />
    </div>
  );
}
