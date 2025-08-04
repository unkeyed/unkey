"use client";

import { LogsProvider } from "../logs/context/logs";
import { LogsControlCloud } from "./_components/control-cloud";
import { LogsControls } from "./_components/controls";
import { ProjectsNavigation } from "./_components/navigation";

export function ProjectsClient() {
  return (
    <div>
      <ProjectsNavigation />
      <LogsProvider>
        <LogsControls />
        <LogsControlCloud />
      </LogsProvider>
      <div>Hello World!</div>
    </div>
  );
}
