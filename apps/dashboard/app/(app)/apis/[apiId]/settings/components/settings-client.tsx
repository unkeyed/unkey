"use client";
import { Separator } from "@/components/ui/separator";
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
      <div className="flex items-center justify-center w-full py-3 ">
        <div className="lg:w-[760px] flex-col justify-center items-center">
          <div className="w-full text-accent-12 font-semibold text-lg pt-[22px] pb-[20px] text-left border-b border-gray-4 px-2">
            API Settings
          </div>
          <div className="flex flex-col gap-6">
            <div>
              <UpdateApiName api={api} />
              <Separator className="bg-gray-4" orientation="horizontal" />
              <CopyApiId apiId={api.id} />
            </div>
            <div>
              <DefaultBytes keyAuth={keyAuth} />
              <Separator className="bg-gray-4" orientation="horizontal" />
              <DefaultPrefix keyAuth={keyAuth} />
            </div>
            <div>
              <UpdateIpWhitelist api={api} workspace={workspace} />
            </div>
            <div>
              <DeleteProtection api={api} />
              <Separator className="bg-gray-4" orientation="horizontal" />
              <DeleteApi api={api} keys={keyAuth.sizeApprox} />
            </div>
          </div>
        </div>
      </div>
    </>
  );
};
