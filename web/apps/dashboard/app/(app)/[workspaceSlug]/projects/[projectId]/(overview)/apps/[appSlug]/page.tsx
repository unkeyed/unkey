"use client";

import { useParams } from "next/navigation";
import { BlankItemPlaceholder } from "../../../components/blank-item-placeholder";
import { useProjectData } from "../../data-provider";

export default function AppPlaceholderPage() {
  const { projectId } = useProjectData();
  const params = useParams();
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  return <BlankItemPlaceholder projectId={projectId} type="app" slug={appSlug} />;
}
