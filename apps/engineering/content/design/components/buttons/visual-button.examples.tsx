"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { VisibleButton } from "@unkey/ui";
import { useState } from "react";

export function VisibleButtonDemo() {
  const [isVisible, setIsVisible] = useState(false);

  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-8">
        <div className="flex items-center gap-4">
          <VisibleButton isVisible={isVisible} setIsVisible={setIsVisible} />
          <span className="text-sm text-gray-600 dark:text-gray-400">
            {isVisible ? "Content is visible" : "Content is hidden"}
          </span>
        </div>
        <div className="flex items-center gap-4">
          <VisibleButton isVisible={isVisible} setIsVisible={setIsVisible} />
          <code className="rounded bg-gray-100 px-2 py-1 text-sm dark:bg-gray-800">
            {isVisible ? "sk_1234567890abcdef" : "••••••••••••••••"}
          </code>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
