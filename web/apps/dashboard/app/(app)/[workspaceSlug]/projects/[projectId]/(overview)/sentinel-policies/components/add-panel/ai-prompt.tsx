"use client";

import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ChevronRight, DoubleChevronRight, Sparkle3 } from "@unkey/icons";
import { Button, FormInput, SlidePanel, toast } from "@unkey/ui";
import { useState } from "react";
import type { KeyLocationFormValues } from "./schema";
import type { PolicyFormValues } from "./schema";

type Props = {
  isOpen: boolean;
  topOffset: number;
  onClose: () => void;
  preview: PolicyFormValues[];
  onPreviewChange: (preview: PolicyFormValues[]) => void;
  onOpenAddPanel: (values: PolicyFormValues, index: number) => void;
};

export function AiPolicyPrompt({
  isOpen,
  topOffset,
  onClose,
  preview,
  onPreviewChange,
  onOpenAddPanel,
}: Props) {
  const [prompt, setPrompt] = useState("");

  const generate = trpc.deploy.environmentSettings.sentinel.generatePolicies.useMutation({
    onSuccess(data) {
      onPreviewChange(data);
    },
    onError(error) {
      toast.error(error.message || "Failed to generate policies", {
        duration: 5000,
        position: "top-right",
      });
    },
  });

  return (
    <SlidePanel.Root isOpen={isOpen} onClose={onClose} topOffset={topOffset}>
      <SlidePanel.Header>
        <div className="flex flex-col">
          <span className="text-gray-12 font-medium text-base leading-8">Generate with AI</span>
          <span className="text-gray-11 text-[13px] leading-5">
            Describe what to protect and how: keyauth and rate limits.
          </span>
        </div>
        <SlidePanel.Close
          aria-label="Close panel"
          className="mt-0.5 inline-flex items-center justify-center size-9 rounded-md hover:bg-grayA-3 transition-colors cursor-pointer"
        >
          <DoubleChevronRight
            iconSize="lg-medium"
            className="text-gray-10 transition-transform duration-300 ease-out"
          />
        </SlidePanel.Close>
      </SlidePanel.Header>

      <SlidePanel.Content>
        <div className="h-full flex flex-col">
          <div className="flex-1 overflow-y-auto bg-grayA-2 flex flex-col gap-5 px-8 pt-6 pb-4">
            <div className="flex flex-col gap-3">
              <div className="flex gap-2 items-end">
                <FormInput
                  placeholder="e.g. bearer keyauth then burst 10/s, sustained 300/min per key"
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  className="flex-1 [&_input]:rounded-md"
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && prompt.trim().length >= 3) {
                      e.preventDefault();
                      generate.mutate({ query: prompt });
                    }
                  }}
                />
                <Button
                  type="button"
                  variant="primary"
                  size="lg"
                  className="shrink-0 rounded-sm"
                  disabled={prompt.trim().length < 3 || generate.isLoading}
                  loading={generate.isLoading}
                  onClick={() => generate.mutate({ query: prompt })}
                >
                  <Sparkle3 iconSize="sm-regular" />
                  Generate
                </Button>
              </div>

              {preview.length > 0 && (
                <div className="border border-grayA-4 rounded-md overflow-hidden">
                  {preview.map((p, i) => (
                    <button
                      key={`${p.name}-${i}`}
                      type="button"
                      onClick={() => onOpenAddPanel(p, i)}
                      className={cn(
                        "flex items-center gap-3 px-4 py-3 w-full text-left hover:bg-grayA-2 transition-colors cursor-pointer",
                        i < preview.length - 1 && "border-b border-grayA-4",
                      )}
                    >
                      <div className="size-6 rounded-full border bg-info-3 border-info-7 text-info-11 text-[11px] font-medium flex items-center justify-center shrink-0">
                        {i + 1}
                      </div>
                      <span className="text-[13px] font-medium text-gray-12 flex-1 min-w-0 truncate">
                        {p.name}
                      </span>
                      <span className="text-xs px-2 py-0.5 rounded-full bg-grayA-2 border border-grayA-4 text-gray-11 shrink-0">
                        {p.type === "keyauth" ? "Key Auth" : "Rate Limit"}
                      </span>
                      {p.type === "ratelimit" && (
                        <span className="text-[13px] text-gray-11 shrink-0 tabular-nums">
                          {p.limit} / {formatWindow(p.windowMs)}
                        </span>
                      )}
                      {p.type === "ratelimit" && (
                        <span className="text-[13px] text-gray-10 shrink-0">
                          {formatKeySource(p.keySource, p.keyValue)}
                        </span>
                      )}
                      {p.type === "keyauth" && (
                        <span className="text-[13px] text-gray-10 shrink-0">
                          {formatLocation(p.locations[0])}
                        </span>
                      )}
                      <ChevronRight
                        iconSize="sm-regular"
                        className="text-gray-9 shrink-0 ml-auto"
                      />
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          {preview.length > 0 && (
            <div className="border-t border-gray-4 bg-white dark:bg-black px-8 py-5 flex items-center justify-end">
              <Button
                type="button"
                variant="primary"
                size="md"
                className="px-3"
                onClick={() => onPreviewChange([])}
              >
                Clear
              </Button>
            </div>
          )}
        </div>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}

function formatWindow(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`;
  }
  if (ms < 60_000) {
    return `${ms / 1000}s`;
  }
  if (ms < 3_600_000) {
    return `${ms / 60_000}min`;
  }
  return `${ms / 3_600_000}h`;
}

function formatLocation(loc: KeyLocationFormValues | undefined): string {
  if (!loc) {
    return "bearer token";
  }
  if (loc.locationType === "bearer") {
    return "bearer token";
  }
  if (loc.locationType === "header") {
    return `header: ${loc.name ?? ""}`;
  }
  return `query: ${loc.name ?? ""}`;
}

function formatKeySource(source: string, value: string): string {
  switch (source) {
    case "remoteIp":
      return "by IP";
    case "authenticatedSubject":
      return "by API key";
    case "principalClaim":
      return `by ${value}`;
    case "header":
      return `by header ${value}`;
    case "path":
      return "by path";
    default:
      return source;
  }
}
