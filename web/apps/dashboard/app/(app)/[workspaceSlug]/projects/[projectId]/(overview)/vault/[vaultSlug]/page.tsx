"use client";

import { useParams } from "next/navigation";
import { BlankItemPlaceholder } from "../../../components/blank-item-placeholder";
import { useProjectData } from "../../data-provider";

export default function VaultPlaceholderPage() {
  const { projectId } = useProjectData();
  const params = useParams();
  const vaultSlug = typeof params?.vaultSlug === "string" ? params.vaultSlug : "";
  return <BlankItemPlaceholder projectId={projectId} type="vault" slug={vaultSlug} />;
}
