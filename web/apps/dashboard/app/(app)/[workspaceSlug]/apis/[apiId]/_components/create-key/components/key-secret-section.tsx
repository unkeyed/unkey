"use client";

import { SecretKey } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/secret-key";
import { CircleInfo } from "@unkey/icons";

type KeySecretSectionProps = {
  keyValue: string;
  apiId: string;
  className?: string;
  secretKeyClassName?: string;
  codeClassName?: string;
};

export const KeySecretSection = ({
  keyValue,
  className,
  secretKeyClassName = "bg-white dark:bg-black",
}: KeySecretSectionProps) => {
  return (
    <div className={className}>
      <div className="flex flex-col gap-2 items-start w-full">
        <div className="text-gray-12 text-sm font-semibold">Key Secret</div>
        <SecretKey value={keyValue} title="API Key" className={secretKeyClassName} />
        <div className="text-gray-9 text-[13px] flex items-center gap-1.5 self-center">
          <CircleInfo className="text-accent-9" iconSize="sm-regular" />
          <span>
            Copy and save this key secret as it won't be shown again.{" "}
            <a
              href="https://www.unkey.com/docs/security/recovering-keys"
              target="_blank"
              rel="noopener noreferrer"
              className="text-info-11 hover:underline"
            >
              Learn more
            </a>
          </span>
        </div>
      </div>
    </div>
  );
};
