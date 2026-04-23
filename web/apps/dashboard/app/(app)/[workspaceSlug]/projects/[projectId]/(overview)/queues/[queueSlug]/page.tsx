"use client";

import { useParams } from "next/navigation";
import { BlankItemPlaceholder } from "../../../components/blank-item-placeholder";
import { useProjectData } from "../../data-provider";

export default function QueuePlaceholderPage() {
  const { projectId } = useProjectData();
  const params = useParams();
  const queueSlug = typeof params?.queueSlug === "string" ? params.queueSlug : "";
  return <BlankItemPlaceholder projectId={projectId} type="queue" slug={queueSlug} />;
}
