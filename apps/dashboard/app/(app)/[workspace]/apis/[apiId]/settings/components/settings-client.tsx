"use client";

import { trpc } from "@/lib/trpc/client";
import { CopyApiId } from "./copy-api-id";
import { DefaultBytes } from "./default-bytes";
import { DefaultPrefix } from "./default-prefix";
import { DeleteApi } from "./delete-api";
import { DeleteProtection } from "./delete-protection";
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

  const { currentApi, keyAuth, workspace } = layoutData;

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
    features: { ipWhitelist: workspace.ipWhitelist },
  };

  return (
    <div className="py-3 w-full flex items-center justify-center">
      <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
        <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          API Settings
        </div>
        <div className="flex flex-col w-full gap-6">
          <div>
            <UpdateApiName api={api} />
            <CopyApiId apiId={api.id} />
          </div>
          <div>
            <DefaultBytes keyAuth={keyAuthForComponents} apiId={api.id} />
            <DefaultPrefix keyAuth={keyAuthForComponents} apiId={api.id} />
          </div>
          <div>
            <UpdateIpWhitelist api={api} workspace={workspaceForComponents} />
          </div>
          <div>
            <DeleteProtection api={api} />
            <DeleteApi api={api} keys={keyAuthForComponents.sizeApprox} />
          </div>
        </div>
      </div>
    </div>
  );
};
