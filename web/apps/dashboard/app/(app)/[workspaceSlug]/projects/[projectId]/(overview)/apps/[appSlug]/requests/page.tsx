"use client";

import { ArrowOppositeDirectionY } from "@unkey/icons";
import { useParams } from "next/navigation";
import { AppSectionPlaceholder } from "../../../../components/app-section-placeholder";
import { useProjectData } from "../../../data-provider";

export default function AppRequestsPage() {
  const { projectId } = useProjectData();
  const params = useParams();
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  return (
    <AppSectionPlaceholder
      projectId={projectId}
      appSlug={appSlug}
      section="Requests"
      icon={ArrowOppositeDirectionY}
    />
  );
}
