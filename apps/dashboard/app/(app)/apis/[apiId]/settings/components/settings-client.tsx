"use client";
import type { Api, KeyAuth, Workspace } from "@unkey/db";
import { CopyApiId } from "./copy-api-id";
import { DefaultBytes } from "./default-bytes";
import { DefaultPrefix } from "./default-prefix";
import { DeleteApi } from "./delete-api";
import { DeleteProtection } from "./delete-protection";
import { UpdateApiName } from "./update-api-name";
import { UpdateIpWhitelist } from "./update-ip-whitelist";

type Props = {
  api: Api;
  workspace: Workspace;
  keyAuth: KeyAuth;
};

export const SettingsClient = ({ api, workspace, keyAuth }: Props) => {
  return (
    <>
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[760px] flex flex-col justify-center items-center gap-5 mx-6">
          <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
            API Settings
          </div>
          <div className="flex flex-col w-full gap-6">
            <div>
              <UpdateApiName api={api} />
              <CopyApiId apiId={api.id} />
            </div>
            <div>
              <DefaultBytes keyAuth={keyAuth} />
              <DefaultPrefix keyAuth={keyAuth} />
            </div>
            <div>
              <UpdateIpWhitelist api={api} workspace={workspace} />
            </div>
            <div>
              <DeleteProtection api={api} />
              <DeleteApi api={api} keys={keyAuth.sizeApprox} />
            </div>
          </div>
        </div>
      </div>
    </>
  );
};
