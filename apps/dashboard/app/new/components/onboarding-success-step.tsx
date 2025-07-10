"use client";
import { SecretKey } from "@/app/(app)/apis/[apiId]/_components/create-key/components/secret-key";
import { ConfirmPopover } from "@/components/confirmation-popover";
import { CircleInfo, TriangleWarning } from "@unkey/icons";
import { Code, CopyButton, VisibleButton } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useRef, useState } from "react";
import { API_ID_PARAM, KEY_PARAM } from "../constants";

type OnboardingSuccessStepProps = {
  isConfirmOpen: boolean;
  setIsConfirmOpen: (open: boolean) => void;
};

export const OnboardingSuccessStep = ({
  isConfirmOpen,
  setIsConfirmOpen,
}: OnboardingSuccessStepProps) => {
  const router = useRouter();
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
  const anchorRef = useRef<HTMLDivElement>(null);
  const searchParams = useSearchParams();

  const apiId = searchParams?.get(API_ID_PARAM);
  const key = searchParams?.get(KEY_PARAM);

  if (!apiId || !key) {
    return (
      <div className="rounded-xl bg-grayA-3 dark:bg-black border border-grayA-3 flex items-center gap-4 px-[22px] py-6">
        <div className="bg-gray-1 size-8 rounded-full flex items-center justify-center flex-shrink-0">
          <TriangleWarning className="text-error-9" size="xl-medium" />
        </div>
        <div className="text-gray-12 text-[13px] leading-6">
          <span className="font-medium">Error:</span> Missing API or key information. Please go back
          and create your API key again to continue with the setup process.
        </div>
      </div>
    );
  }

  const split = key.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);

  const snippet = `curl -XPOST '${
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"
  }/v1/keys.verifyKey' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${key}",
    "apiId": "${apiId}"
  }'`;

  return (
    <>
      <div>
        <span className="text-gray-11 text-[13px] leading-6" ref={anchorRef}>
          Run this command to verify your new API key against the API ID. This ensures your key is
          ready for authenticated requests.
        </span>
        <div className="flex flex-col gap-2 items-start w-full mt-6">
          <div className="text-gray-12 text-sm font-medium">Key Secret</div>
          <SecretKey value={key} title="API Key" className="bg-gray-2" />
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
          <div className="text-gray-12 text-sm font-medium">Try It Out</div>
          <Code
            className="bg-gray-2"
            visibleButton={
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
            }
            copyButton={<CopyButton value={snippet} />}
          >
            {showKeyInSnippet ? snippet : snippet.replace(key, maskedKey)}
          </Code>
        </div>
      </div>
      <ConfirmPopover
        isOpen={isConfirmOpen}
        onOpenChange={setIsConfirmOpen}
        onConfirm={() => {
          setIsConfirmOpen(false);

          router.push("/apis");
        }}
        triggerRef={anchorRef}
        title="You won't see this secret key again!"
        description="Make sure to copy your secret key before closing. It cannot be retrieved later."
        confirmButtonText="Close anyway"
        cancelButtonText="Dismiss"
        variant="warning"
        popoverProps={{
          side: "right",
          align: "end",
          sideOffset: 5,
          alignOffset: 30,
          onOpenAutoFocus: (e) => e.preventDefault(),
        }}
      />
    </>
  );
};
