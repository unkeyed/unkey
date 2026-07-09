"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import { trpc } from "@/lib/trpc/client";
import { Plus } from "@unkey/icons";
import { CreateKeyDialog } from "./_components/create-key";

export function CreateKeyAction({ apiId }: { apiId: string }) {
  const { data } = trpc.api.queryApiKeyDetails.useQuery(
    { apiId },
    { enabled: Boolean(apiId), retry: 3, retryDelay: 1000 },
  );

  if (!data?.keyAuth) {
    return (
      <NavbarActionButton disabled>
        <Plus />
        Create key
      </NavbarActionButton>
    );
  }

  return (
    <CreateKeyDialog
      keyspaceId={data.keyAuth.id}
      apiId={apiId}
      copyIdValue={apiId}
      keyspaceDefaults={data.currentApi.keyspaceDefaults}
    />
  );
}
