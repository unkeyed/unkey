"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { VisibleButton } from "@unkey/ui";
import { useState } from "react";

export function VisibleButtonDemo() {
  const [isVisible, setIsVisible] = useState(false);

  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-8">
        <div className="mx-auto">Default Variant</div>
        <div className="rounded flex items-center gap-2 border border-gray-6 mx-auto text-center">
          <code className="flex-row px-2 bg-transparent border-none mx-2 overflow-hidden text-ellipsis">
            {isVisible ? "Content is visible" : "Content is hidden"}
          </code>
          <VisibleButton
            isVisible={isVisible}
            setIsVisible={setIsVisible}
            className="right-1"
            title="Password"
            variant="outline"
          />
        </div>
        <div className="mx-auto">Ghost Variant</div>
        <div className="rounded flex items-center gap-2 border border-gray-6  mx-auto text-center">
          <code className="flex-2 px-2 bg-transparent border-none overflow-hidden text-ellipsis">
            {isVisible ? "sk_1234567890abcdef" : "••••••••••••••••"}
          </code>
          <VisibleButton
            isVisible={isVisible}
            setIsVisible={setIsVisible}
            className="right-1 focus:ring-0"
            variant="ghost"
            title="Key"
          />
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
