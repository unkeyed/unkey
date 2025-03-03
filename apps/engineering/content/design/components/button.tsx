"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button } from "@unkey/ui";
import { SearchIcon } from "lucide-react";

export const ButtonWithKeyboardShortcut = () => {
  return (
    <RenderComponentWithSnippet>
      <Button
        variant="primary"
        keyboard={{
          display: "⌘K",
          trigger: (e) => (e.metaKey || e.ctrlKey) && e.key === "k",
          callback: () => console.log("Command+K pressed!"),
        }}
      >
        <SearchIcon />
        <span>Search</span>
      </Button>
    </RenderComponentWithSnippet>
  );
};
