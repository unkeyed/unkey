"use client";

import { Check, Clipboard, Layers2 } from "@unkey/icons";
import { InfoTooltip, toast } from "@unkey/ui";
import { useState } from "react";

export function ImageSource({
  image,
  copyValue = image,
}: { image: string | null; copyValue?: string | null }) {
  const [copied, setCopied] = useState(false);

  if (!image) {
    return (
      <InfoTooltip
        content="No source info"
        variant="inverted"
        position={{ side: "top", align: "start" }}
      >
        <span className="flex items-center gap-1 min-w-0">
          <Layers2 iconSize="sm-regular" className="text-accent-12 shrink-0" />
          <span className="font-mono text-xs text-accent-12">unknown</span>
        </span>
      </InfoTooltip>
    );
  }

  return (
    <InfoTooltip
      content={copyValue && copyValue !== image ? `Resolved image: ${copyValue}` : image}
      variant="inverted"
      position={{ side: "top", align: "start" }}
      asChild
    >
      <button
        type="button"
        aria-label={`Copy image source: ${copyValue ?? image}`}
        className="group flex items-center gap-1 min-w-0"
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(copyValue ?? image);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
            toast.success("Source copied to clipboard");
          } catch {
            toast.error("Failed to copy source");
          }
        }}
      >
        <Layers2 iconSize="sm-regular" className="text-accent-12 shrink-0" />
        <span className="font-mono text-xs text-accent-12 truncate max-w-48">{image}</span>
        {copied ? (
          <Check iconSize="sm-regular" className="text-success-11 shrink-0" />
        ) : (
          <Clipboard
            iconSize="sm-regular"
            className="text-gray-9 shrink-0 opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100"
          />
        )}
      </button>
    </InfoTooltip>
  );
}
