"use client";

import { PROMPT } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/agent-setup";
import {
  CopyIcon,
  TerminalIcon,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { Button } from "@unkey/ui";
import { useState } from "react";

// Deploying an app is one of two onboarding paths (the other is "connect a
// repo"), not a generic workspace-level nudge — so this lives inline wherever
// a project asks "how do I get code in here" rather than as its own card.
function useCopyPrompt() {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(PROMPT);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1800);
  };
  return { copied, copy };
}

export function CopyAgentPromptButton() {
  const { copied, copy } = useCopyPrompt();
  return (
    <Button size="md" variant="outline" onClick={copy}>
      <CopyIcon className="size-3.5" />
      {copied ? "Copied" : "Copy agent prompt"}
    </Button>
  );
}

export function AgentConnectTile() {
  const { copied, copy } = useCopyPrompt();
  return (
    <button
      type="button"
      onClick={copy}
      className="flex flex-col gap-2 rounded-lg border border-grayA-4 p-3 text-left hover:border-grayA-7"
    >
      <TerminalIcon className="size-4 text-gray-9" />
      <div>
        <div className="text-[13px] font-medium text-accent-12">Set up your agent</div>
        <div className="mt-0.5 text-xs text-gray-9">
          {copied ? "Copied — paste into your agent" : "Claude, Cursor, Codex"}
        </div>
      </div>
    </button>
  );
}
