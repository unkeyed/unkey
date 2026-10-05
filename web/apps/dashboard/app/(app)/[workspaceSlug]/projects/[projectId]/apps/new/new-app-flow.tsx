"use client";

import { FlowLoader, viewportHeight } from "./flow";
import { FlowColumn } from "./flow-column";

export function NewAppFlow({ projectId }: { projectId: string }) {
  return (
    <FlowLoader projectId={projectId}>
      <div className="flex flex-col bg-background" style={viewportHeight}>
        <FlowColumn />
      </div>
    </FlowLoader>
  );
}
