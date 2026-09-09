"use client";

import { CopyButton, Skeleton, Tabs, TabsContent, TabsList, TabsTrigger, toast } from "@unkey/ui";
import { IconClockOutline12, IconCloneOutline12, IconLink4Outline12 } from "nucleo-ui-outline-12";
import { useState } from "react";
import { trpc } from "@/lib/trpc/client";
import { KeySecret } from "./key-secret-section";

type KeyPayloadTabsProps = {
  keyValue: string;
};

export function KeyPayloadTabs({ keyValue }: KeyPayloadTabsProps) {
  const [link, setLink] = useState<string | null>(null);
  const [hasError, setHasError] = useState(false);

  const create = trpc.share.create.useMutation({
    onSuccess: ({ url }) => {
      setLink(url);
      setHasError(false);
    },
    onError: (err) => {
      console.error(err);
      setHasError(true);
      toast.error("Could not create a secure link. Please try again.");
    },
  });

  const isGenerating = create.isLoading;
  const generate = () => {
    setHasError(false);
    create.mutate({ secret: keyValue });
  };

  // Generate lazily the first time the Secure link tab becomes active, and never
  // re-fire on subsequent activations so the same link stays cached.
  const handleValueChange = (value: string) => {
    if (value === "secure-link" && !link && !isGenerating && !hasError) {
      generate();
    }
  };

  return (
    <Tabs
      defaultValue="copy-secret"
      onValueChange={handleValueChange}
      className="w-full flex flex-col items-center gap-4"
    >
      <TabsList className="w-full bg-grayA-3 h-10">
        <TabsTrigger value="copy-secret" className="flex-1 gap-2 hover:bg-grayA-3">
          <IconCloneOutline12 />
          Key secret
        </TabsTrigger>
        <TabsTrigger value="secure-link" className="flex-1 gap-2 hover:bg-grayA-3">
          <IconLink4Outline12 />
          Secure link
        </TabsTrigger>
      </TabsList>

      <TabsContent value="copy-secret" className="w-full mt-0 min-h-[76px]">
        <KeySecret keyValue={keyValue} />
      </TabsContent>

      <TabsContent value="secure-link" className="w-full mt-0 min-h-[76px]">
        <div className="w-full flex flex-col gap-2 items-start">
          {hasError ? (
            <div className="w-full text-center py-2 text-[13px] text-gray-9">
              <span className="text-warning-11">Could not create a secure link.</span>{" "}
              <button
                type="button"
                onClick={generate}
                disabled={isGenerating}
                className="text-info-11 hover:underline disabled:opacity-50 disabled:no-underline"
              >
                Retry
              </button>
            </div>
          ) : isGenerating ? (
            <Skeleton className="w-full h-[42px] rounded-xl" />
          ) : link ? (
            <>
              <div className="w-full px-4 py-2 bg-white dark:bg-black border rounded-xl border-grayA-5">
                <div className="flex items-center justify-between w-full gap-3">
                  <IconLink4Outline12 className="text-gray-12 shrink-0" />
                  <p className="flex-1 min-w-0 truncate font-mono text-[13px] text-grayA-12">
                    {link}
                  </p>
                  <div className="flex items-center shrink-0">
                    <CopyButton value={link} title="Copy secure link" />
                  </div>
                </div>
              </div>
              <div className="text-gray-9 text-[13px] flex items-center gap-1.5 self-center">
                <IconClockOutline12 className="text-primary" />
                <span>
                  Expires after 72hrs.
                  <button
                    type="button"
                    onClick={generate}
                    disabled={isGenerating}
                    className="text-info-11 hover:underline disabled:opacity-50 disabled:no-underline ml-1"
                  >
                    Regenerate
                  </button>
                </span>
              </div>
            </>
          ) : null}
        </div>
      </TabsContent>
    </Tabs>
  );
}
