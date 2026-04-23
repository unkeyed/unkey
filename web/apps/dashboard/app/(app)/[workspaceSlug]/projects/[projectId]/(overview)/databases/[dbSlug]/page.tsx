"use client";

import { useParams } from "next/navigation";
import { BlankItemPlaceholder } from "../../../components/blank-item-placeholder";
import { useProjectData } from "../../data-provider";

export default function DatabasePlaceholderPage() {
  const { projectId } = useProjectData();
  const params = useParams();
  const dbSlug = typeof params?.dbSlug === "string" ? params.dbSlug : "";
  return <BlankItemPlaceholder projectId={projectId} type="database" slug={dbSlug} />;
}
