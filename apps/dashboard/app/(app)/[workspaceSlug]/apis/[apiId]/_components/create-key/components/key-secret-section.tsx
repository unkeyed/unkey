"use client";

import { SecretKey } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/secret-key";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { CircleInfo } from "@unkey/icons";
import { Code, CopyButton, VisibleButton } from "@unkey/ui";
import { useState } from "react";

type KeySecretSectionProps = {
  keyValue: string;
  apiId: string;
  className?: string;
  secretKeyClassName?: string;
  codeClassName?: string;
};

export const KeySecretSection = ({
  keyValue,
  apiId,
  className,
  secretKeyClassName = "bg-white dark:bg-black",
  codeClassName,
}: KeySecretSectionProps) => {
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);

  const split = keyValue.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);

  const snippet = `curl -XPOST '${
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.com"
  }/v2/keys.verifyKey' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${keyValue}"
  }'`;

  return (
    <div className={className}>
      <div className="flex flex-col gap-2 items-start w-full">
        <div className="text-gray-12 text-sm font-semibold">Key Secret</div>
        <SecretKey
          value={keyValue}
          title="API Key"
          className={secretKeyClassName}
        />
        <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
          <CircleInfo className="text-accent-9" iconsize="sm-regular" />
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
      <div className="flex flex-col gap-2 items-start w-full mt-8">
        <div className="text-gray-12 text-sm font-semibold">Try It Out</div>
        <div className="relative w-full">
          <Code className={codeClassName} preClassName="overflow-x-auto p-0 mb-0">
            <div className="p-4">
              {showKeyInSnippet ? snippet : snippet.replace(keyValue, maskedKey)}
            </div>
          </Code>
          <VisibleButton
            isVisible={showKeyInSnippet}
            setIsVisible={setShowKeyInSnippet}
            className="absolute right-12 top-3 md:top-4"
          />
          <CopyButton value={snippet} className="absolute right-3 md:right-4 top-3 md:top-4" />
        </div>
        <Alert variant="warn">
          <div className="flex items-start mb-1 gap-2">
            <CircleInfo
              iconsize="lg-regular"
              aria-hidden="true"
              className="flex-shrink-0"
            />
            <div>
              <AlertTitle className="mb-1">Root Key Required</AlertTitle>
              <AlertDescription className="text-gray-12">
                To verify keys, you'll need a root key with{" "}
                <code className="bg-gray-3 px-1 rounded text-xs">
                  api.*.verify_key
                </code>{" "}
                or{" "}
                <code className="bg-gray-3 px-1 rounded text-xs">
                  api.{apiId}.verify_key
                </code>{" "}
                permission.
                <br />
                <a
                  href="https://www.unkey.com/docs/security/root-keys"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-info-11 hover:underline"
                >
                  Learn more about root keys
                </a>
              </AlertDescription>
            </div>
          </div>
        </Alert>
      </div>
    </div>
  );
};
