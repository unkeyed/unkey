"use client";

import { SecretKey } from "@/app/(app)/apis/[apiId]/_components/create-key/components/secret-key";
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
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"
  }/v1/keys.verifyKey' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${keyValue}",
    "apiId": "${apiId}"
  }'`;

  return (
    <div className={className}>
      <div className="flex flex-col gap-2 items-start w-full">
        <div className="text-gray-12 text-sm font-semibold">Key Secret</div>
        <SecretKey value={keyValue} title="API Key" className={secretKeyClassName} />
        <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
          <CircleInfo className="text-accent-9" size="sm-regular" />
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
        <Code
          className={codeClassName}
          visibleButton={
            <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
          }
          copyButton={<CopyButton value={snippet} />}
        >
          {showKeyInSnippet ? snippet : snippet.replace(keyValue, maskedKey)}
        </Code>
      </div>
    </div>
  );
};
