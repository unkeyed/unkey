"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useApiName } from "@/hooks/use-api-name";
import { use } from "react";
import { PortalLifecyclePage } from "./components/portal-lifecycle-page";

type Props = {
  params: Promise<{ apiId: string }>;
};

export default function ApiPortalPage(props: Props) {
  const { apiId } = use(props.params);
  const { name, isLoading: nameLoading } = useApiName(apiId);
  const { keyAuthId, isLoading: keyAuthIdLoading } = useApiKeyAuthId(apiId);

  return (
    <PortalLifecyclePage
      resourceName={name ?? "API"}
      keyAuthId={keyAuthId}
      // The configuration view seeds editable state from the API name, so a
      // still-resolving name keeps the surface in its loading state rather than
      // rendering with a placeholder that sticks.
      keyAuthIdLoading={nameLoading || keyAuthIdLoading}
    />
  );
}
