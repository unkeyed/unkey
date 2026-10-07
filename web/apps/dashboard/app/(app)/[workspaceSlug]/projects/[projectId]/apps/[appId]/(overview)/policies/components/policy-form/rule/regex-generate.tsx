"use client";

import { trpc } from "@/lib/trpc/client";
import { IconSparkle3Outline18 } from "@unkey/icons";
import { Button, Input, toast } from "@unkey/ui";
import { useState } from "react";

export function RegexGenerate({
  conditionType,
  placeholder,
  onGenerated,
}: {
  conditionType: "path" | "header" | "queryParam";
  placeholder: string;
  onGenerated: (pattern: string) => void;
}) {
  const [prompt, setPrompt] = useState("");
  const generateRegex = trpc.deploy.environmentSettings.policies.generateRegex.useMutation({
    onSuccess(data) {
      onGenerated(data.pattern);
      setPrompt("");
    },
    onError(error) {
      toast.error(error.message || "Failed to generate regex pattern", {
        duration: 5000,
        position: "top-right",
      });
    },
  });
  const canGenerate = prompt.trim().length >= 3 && !generateRegex.isLoading;
  const generate = () => {
    if (canGenerate) {
      generateRegex.mutate({ query: prompt, conditionType });
    }
  };

  return (
    <div className="flex items-center gap-2">
      <Input
        aria-label="Describe what to match"
        className="h-8 min-w-0 flex-1 text-xs"
        placeholder={placeholder}
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            generate();
          }
        }}
      />
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="shrink-0"
        disabled={!canGenerate}
        loading={generateRegex.isLoading}
        onClick={generate}
      >
        <IconSparkle3Outline18 />
        Generate regex
      </Button>
    </div>
  );
}
