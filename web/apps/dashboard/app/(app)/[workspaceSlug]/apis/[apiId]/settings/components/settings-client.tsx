"use client";

import { trpc } from "@/lib/trpc/client";
import {
  SettingsDangerZone,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
} from "@unkey/ui";
import { CopyApiId } from "./copy-api-id";
import { CopyKeySpaceId } from "./copy-key-space-id";
import { DefaultBytes } from "./default-bytes";
import { DefaultPrefix } from "./default-prefix";
import { DeleteApi } from "./delete-api";
import { DeleteProtection } from "./delete-protection";
import { SettingsClientSkeleton } from "./skeleton";
import { UpdateApiName } from "./update-api-name";

export const SettingsClient = ({ apiId }: { apiId: string }) => {
  const {
    data: layoutData,
    isLoading,
    error,
  } = trpc.api.queryApiKeyDetails.useQuery({
    apiId,
  });

  if (isLoading) {
    return <SettingsClientSkeleton />;
  }

  if (error) {
    throw new Error(`Failed to fetch settings data: ${error.message}`);
  }

  if (!layoutData || !layoutData.keyAuth) {
    throw new Error("KeyAuth configuration not found");
  }

  const { currentApi, keyAuth } = layoutData;

  const api = {
    id: currentApi.id,
    name: currentApi.name,
    workspaceId: currentApi.workspaceId,
    deleteProtection: currentApi.deleteProtection,
  };

  const keyAuthForComponents = {
    id: keyAuth.id,
    defaultPrefix: keyAuth.defaultPrefix,
    defaultBytes: keyAuth.defaultBytes,
    sizeApprox: keyAuth.sizeApprox,
  };

  return (
    <>
      <SettingsGroup>
        <SettingsGroupTitle>General</SettingsGroupTitle>
        <SettingsGroupContent>
          <UpdateApiName api={api} />
          <CopyApiId apiId={api.id} />
          <CopyKeySpaceId keySpaceId={keyAuth.id} />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsGroup>
        <SettingsGroupTitle>Key defaults</SettingsGroupTitle>
        <SettingsGroupContent>
          <DefaultBytes keyAuth={keyAuthForComponents} apiId={api.id} />
          <DefaultPrefix keyAuth={keyAuthForComponents} apiId={api.id} />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsDangerZone>
        <DeleteProtection api={api} />
        <DeleteApi api={api} keys={keyAuthForComponents.sizeApprox} />
      </SettingsDangerZone>
    </>
  );
};
