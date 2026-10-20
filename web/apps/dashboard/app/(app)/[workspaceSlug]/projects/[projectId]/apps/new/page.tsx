"use client";

import { useParams } from "next/navigation";
import { FlowLoader } from "./flow";
import { FlowColumn } from "./flow-column";

export default function NewAppPage() {
  const { projectId } = useParams<{ projectId: string }>();
  return (
    <FlowLoader projectId={projectId}>
      <FlowColumn />
    </FlowLoader>
  );
}
