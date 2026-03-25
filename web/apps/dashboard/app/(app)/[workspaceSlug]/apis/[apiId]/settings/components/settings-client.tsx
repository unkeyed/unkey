"use client";

import { trpc } from "@/lib/trpc/client";
import { SettingCardGroup, SettingsDangerZone } from "@unkey/ui";
import { CopyApiId } from "./copy-api-id";
import { CopyKeySpaceId } from "./copy-key-space-id";
import { DefaultBytes } from "./default-bytes";
import { DefaultPrefix } from "./default-prefix";
import { DeleteApi } from "./delete-api";
import { DeleteProtection } from "./delete-protection";
import { Shell } from "./shell";
import { SettingsClientSkeleton } from "./skeleton";
import { UpdateApiName } from "./update-api-name";
import { UpdateIpWhitelist } from "./update-ip-whitelist";

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

  const { currentApi, keyAuth, workspace: workspaceData } = layoutData;

  const api = {
    id: currentApi.id,
    name: currentApi.name,
    workspaceId: currentApi.workspaceId,
    deleteProtection: currentApi.deleteProtection,
    ipWhitelist: currentApi.ipWhitelist,
  };

  const keyAuthForComponents = {
    id: keyAuth.id,
    defaultPrefix: keyAuth.defaultPrefix,
    defaultBytes: keyAuth.defaultBytes,
    sizeApprox: keyAuth.sizeApprox,
  };

  const workspaceForComponents = {
    features: { ipWhitelist: workspaceData.ipWhitelist },
  };

  return (
    <Shell>
      <div className="flex flex-col gap-2 items-center">
        <span className="font-semibold text-gray-12 leading-8 text-lg">API Settings</span>
        <span className="leading-4 text-gray-11 text-[13px]">
          Configure your API name, default key settings, and access controls.
        </span>
      </div>
      <div className="w-full">
        <SettingCardGroup>
          <UpdateApiName api={api} />
          <CopyApiId apiId={api.id} />
          <CopyKeySpaceId keySpaceId={keyAuth.id} />
        </SettingCardGroup>
      </div>
      <div className="w-full">
        <SettingCardGroup>
          <DefaultBytes keyAuth={keyAuthForComponents} apiId={api.id} />
          <DefaultPrefix keyAuth={keyAuthForComponents} apiId={api.id} />
        </SettingCardGroup>
      </div>
      <div className="w-full">
        <SettingCardGroup>
          <UpdateIpWhitelist api={api} workspace={workspaceForComponents} />
        </SettingCardGroup>
      </div>
      <SettingsDangerZone>
        <DeleteProtection api={api} />
        <DeleteApi api={api} keys={keyAuthForComponents.sizeApprox} />
      </SettingsDangerZone>
    </Shell>
  );
};
