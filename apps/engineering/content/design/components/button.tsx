"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button } from "@unkey/ui";
import { useState } from "react";

export const ButtonWithKeyboardShortcut = () => {
  const [count, setCount] = useState(0);
  const [isLoading, setIsLoading] = useState(false);

  const handleClick = async () => {
    setIsLoading(true);
    await new Promise((resolve) => setTimeout(resolve, 1000));
    setCount((prev) => prev + 1);
    setIsLoading(false);
  };

  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-4 w-full max-w-md">
        <div className="flex items-center gap-4">
          <Button
            onClick={handleClick}
            loading={isLoading}
            keyboard={{
              display: "⌘B",
              trigger: (e) => (e.metaKey || e.ctrlKey) && e.key === "b",
              callback: handleClick,
            }}
          >
            Increment Counter
          </Button>
          <span className="text-sm">Count: {count}</span>
        </div>
        <p className="text-sm text-gray-11">
          Press <kbd className="px-1.5 py-0.5 rounded bg-gray-3 border border-gray-6">⌘B</kbd> to
          increment the counter
        </p>
      </div>
    </RenderComponentWithSnippet>
  );
};
