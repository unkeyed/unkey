"use client";

import { useApiName } from "@/hooks/use-api-name";
import { use } from "react";
import { PortalLifecyclePage } from "./components/portal-lifecycle-page";

type Props = {
  params: Promise<{ apiId: string }>;
};

export default function ApiPortalPage(props: Props) {
  const { apiId } = use(props.params);
  const { name, isLoading } = useApiName(apiId);

  // The config page seeds editable branding state from resourceName, so wait
  // for a real name instead of rendering with a placeholder that sticks.
  if (isLoading) {
    return null;
  }

  return <PortalLifecyclePage resourceId={apiId} resourceName={name ?? "API"} />;
}
