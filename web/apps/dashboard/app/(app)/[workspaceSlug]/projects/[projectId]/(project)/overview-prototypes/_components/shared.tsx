"use client";

import type { ProjectOverview } from "@/lib/trpc/routers/deploy/project/overview";
import { IconBook2Outline18, IconChatsOutline18, IconSquareTerminalOutline18 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { type ReactNode, useState } from "react";
import { compact } from "../../overview/_components/overview-model";
import type { OverviewModel } from "../../overview/_components/overview-model";
import type { CanvasActions, CanvasLinks, MenuEntry } from "./proto-canvas";
import type { Totals } from "./totals";

export type VariantProps = {
  data: ProjectOverview;
  model: OverviewModel;
  links: CanvasLinks;
  actions: CanvasActions;
  title: string;
  crumb: string;
  totals: Totals;
};

const AGENT_PROMPT =
  "Set up Unkey in my project. Fetch https://unkey.com/agent/setup.md and follow it.";

export const HELP_LINKS = [
  {
    href: "https://unkey.com/discord",
    icon: <IconChatsOutline18 />,
    title: "Ask in Discord",
    description: "Community answers",
  },
  {
    href: "mailto:support@unkey.dev",
    icon: <IconChatsOutline18 />,
    title: "Get support",
    description: "Talk to an engineer",
  },
  {
    href: "https://unkey.com/docs",
    icon: <IconBook2Outline18 />,
    title: "Documentation",
    description: "Guides and API reference",
  },
];

export function useCopyPrompt() {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(AGENT_PROMPT);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1800);
  };
  return { copied, copy };
}

export function helpEntries(copy: () => void): MenuEntry[] {
  return [
    {
      label: "Set up with your agent",
      description: "Copy a prompt for Claude, Cursor, Codex",
      icon: <IconSquareTerminalOutline18 />,
      run: copy,
    },
    ...HELP_LINKS.map((l) => ({
      label: l.title,
      description: l.description,
      icon: l.icon,
      run: () => window.open(l.href, "_blank", "noreferrer"),
    })),
  ];
}

export function HelpRow() {
  const { copied, copy } = useCopyPrompt();
  const tile =
    "flex flex-col gap-2 rounded-lg border border-border bg-raised p-3 text-left transition-colors hover:border-strong";
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-[13px] font-medium text-gray-12">Need help?</h2>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <button type="button" onClick={copy} className={tile}>
          <IconSquareTerminalOutline18 className="size-4 text-gray-9" />
          <span>
            <span className="block text-[13px] font-medium text-gray-12">
              Set up with your agent
            </span>
            <span className="block text-xs text-gray-9">
              {copied ? "Prompt copied. Paste it into your agent." : "Claude, Cursor, Codex"}
            </span>
          </span>
        </button>
        {HELP_LINKS.map((l) => (
          <a key={l.title} href={l.href} target="_blank" rel="noreferrer" className={tile}>
            <span className="text-gray-9 [&_svg]:size-4">{l.icon}</span>
            <span>
              <span className="block text-[13px] font-medium text-gray-12">{l.title}</span>
              <span className="block text-xs text-gray-9">{l.description}</span>
            </span>
          </a>
        ))}
      </div>
    </section>
  );
}

export function HelpLine({ className }: { className?: string }) {
  const { copied, copy } = useCopyPrompt();
  return (
    <div
      className={cn("flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-9", className)}
    >
      <span>Stuck?</span>
      <button type="button" onClick={copy} className="text-gray-11 hover:text-gray-12">
        {copied ? "Prompt copied" : "Copy agent prompt"}
      </button>
      {HELP_LINKS.map((l) => (
        <a
          key={l.title}
          href={l.href}
          target="_blank"
          rel="noreferrer"
          className="text-gray-11 hover:text-gray-12"
        >
          {l.title}
        </a>
      ))}
    </div>
  );
}

export function num(n: number | undefined): string {
  return n == null ? "…" : compact(n);
}

export function pct(n: number | undefined): string {
  return n == null ? "…" : `${n.toFixed(1)}%`;
}

export function healthLine(t: Totals, apps: number): string {
  if (apps === 0) {
    return "No apps deployed";
  }
  const parts = [`${t.ready} ready`];
  if (t.building) {
    parts.push(`${t.building} building`);
  }
  if (t.waiting) {
    parts.push(`${t.waiting} awaiting approval`);
  }
  if (t.failed) {
    parts.push(`${t.failed} failed`);
  }
  return parts.join(" · ");
}

export function HealthDot({ totals }: { totals: Totals }) {
  return (
    <span
      className={cn(
        "size-1.5 shrink-0 rounded-full",
        totals.failed ? "bg-error-9" : totals.waiting ? "bg-warning-9" : "bg-success-9",
      )}
    />
  );
}

export function MonoLabel({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span
      className={cn("font-mono text-[10px] uppercase tracking-[0.08em] text-gray-9", className)}
    >
      {children}
    </span>
  );
}
