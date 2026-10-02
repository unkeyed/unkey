"use client";

import {
  IconCheckOutline12,
  IconChevronDownOutline12,
  IconCircleXmarkOutline12,
  IconCloneOutline12,
  IconDoubleChevronDownOutline12,
  IconMagnifierOutline12,
  type IconProps,
  IconTriangleWarningOutline12,
  IconXmarkOutline12,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, InfoTooltip, Loading, toast } from "@unkey/ui";
import { cn } from "cn";
import {
  type ComponentProps,
  type ComponentType,
  type ReactNode,
  type Ref,
  useEffect,
  useState,
} from "react";
import {
  type HeaderLabel,
  type JumpTone,
  type ScrollToLatest,
  type ToneCounter,
  headerLabelText,
} from "./build-logs-header-state";

const TONE_TOOLTIPS: Record<JumpTone, string> = { error: "Errors", warning: "Warnings" };

const TONE_ICONS: Record<JumpTone, { Icon: ComponentType<IconProps>; activeClass: string }> = {
  error: { Icon: IconCircleXmarkOutline12, activeClass: "text-error-11" },
  warning: { Icon: IconTriangleWarningOutline12, activeClass: "text-warning-11" },
};

type Props = {
  label: HeaderLabel;
  copyText: () => string;
  counters: ToneCounter[];
  scrollToLatest: ScrollToLatest;
  onScrollToLatest: () => void;
  onToggleTone: (tone: JumpTone) => void;
  onStepTone: (tone: JumpTone, delta: 1 | -1) => void;
  query: string;
  onQueryChange: (query: string) => void;
  searchRef: Ref<HTMLInputElement>;
  searchShortcutLabel: string;
};

export function BuildLogsHeader({
  label,
  copyText,
  counters,
  scrollToLatest,
  onScrollToLatest,
  onToggleTone,
  onStepTone,
  query,
  onQueryChange,
  searchRef,
  searchShortcutLabel,
}: Props) {
  return (
    <div className="flex h-11 items-center justify-between gap-4 border-b border-grayA-3 px-4">
      <div className="flex min-w-0 items-center gap-2">
        <CopyLinesButton label={headerLabelText(label)} text={copyText} />
      </div>
      <div className="flex items-center gap-2">
        {scrollToLatest.type !== "hidden" && (
          <InfoTooltip content="Scroll to latest" asChild position={{ side: "top" }}>
            <Button
              variant="outline"
              size="icon"
              aria-label="Scroll to latest"
              className="size-7 text-gray-11 hover:text-gray-12 [&_svg]:size-2.5"
              loading={scrollToLatest.type === "pending"}
              onClick={onScrollToLatest}
            >
              <IconDoubleChevronDownOutline12 />
            </Button>
          </InfoTooltip>
        )}
        <div className="flex h-7 items-stretch overflow-hidden rounded-md border border-grayA-5 bg-gray-1 text-xs has-[input:focus]:border-grayA-8">
          {counters.map((counter) => (
            <ToneCounterSegment
              key={counter.tone}
              counter={counter}
              onToggle={() => onToggleTone(counter.tone)}
              onStep={(delta) => onStepTone(counter.tone, delta)}
            />
          ))}
          <div className="relative flex items-center">
            <IconMagnifierOutline12 className="pointer-events-none absolute left-2 size-3 text-gray-9" />
            <input
              ref={searchRef}
              type="search"
              value={query}
              placeholder="Find in logs"
              aria-label="Find in logs"
              className="h-full w-44 bg-transparent pl-6 pr-9 text-gray-12 placeholder:text-grayA-8 focus-visible:outline-hidden [&::-webkit-search-cancel-button]:hidden"
              onChange={(event) => onQueryChange(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Escape") {
                  onQueryChange("");
                  event.currentTarget.blur();
                }
              }}
            />
            <kbd className="pointer-events-none absolute right-1.5 rounded-sm border border-grayA-4 bg-grayA-2 px-1 font-sans text-[10px] leading-4 text-gray-10 max-md:hidden">
              {searchShortcutLabel}
            </kbd>
          </div>
        </div>
      </div>
    </div>
  );
}

function ToneCounterSegment({
  counter,
  onToggle,
  onStep,
}: {
  counter: ToneCounter;
  onToggle: () => void;
  onStep: (delta: 1 | -1) => void;
}) {
  const { tone, count, hasMore } = counter;
  const { Icon, activeClass } = TONE_ICONS[tone];
  const isActive = counter.type === "active";

  return (
    <div
      className={cn(
        "flex items-stretch border-r border-grayA-5 transition-colors duration-150 ease motion-reduce:transition-none",
        isActive && "bg-grayA-3",
      )}
    >
      <SegmentButton
        disabled={count === 0 && !hasMore}
        aria-pressed={isActive}
        aria-label={headerLabelText({ type: "tone", tone, lines: count })}
        tooltip={TONE_TOOLTIPS[tone]}
        className="gap-1 px-2"
        onClick={onToggle}
      >
        {match(counter)
          .with({ type: "pending" }, () => <Loading size={12} />)
          .with({ type: "idle" }, () => <Icon />)
          .with({ type: "active" }, () => <Icon className={activeClass} />)
          .exhaustive()}
        <span className="flex">
          <Reveal open={isActive}>
            {`${((counter.type === "active" ? counter.position : 0) + 1).toLocaleString()}/`}
          </Reveal>
          {count.toLocaleString()}
          {hasMore && "+"}
        </span>
      </SegmentButton>
      <Reveal open={isActive} className="items-stretch">
        {([-1, 1] as const).map((delta) => (
          <SegmentButton
            key={delta}
            tooltip={delta === 1 ? `Next ${tone}` : `Previous ${tone}`}
            className="px-1"
            onClick={() => onStep(delta)}
          >
            <IconChevronDownOutline12 className={cn(delta === -1 && "rotate-180")} />
          </SegmentButton>
        ))}
        <SegmentButton
          tooltip="Clear"
          className="ml-0.5 border-l border-grayA-5 px-1.5"
          onClick={onToggle}
        >
          <IconXmarkOutline12 />
        </SegmentButton>
      </Reveal>
    </div>
  );
}

function SegmentButton({
  tooltip,
  className,
  ...props
}: ComponentProps<"button"> & { tooltip: string }) {
  return (
    <InfoTooltip content={tooltip} asChild position={{ side: "top" }}>
      <button
        type="button"
        aria-label={tooltip}
        className={cn(
          "flex items-center text-gray-11 tabular-nums hover:bg-grayA-2 hover:text-gray-12 disabled:pointer-events-none disabled:opacity-50 focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-gray-7 [&_svg]:size-3",
          className,
        )}
        {...props}
      />
    </InfoTooltip>
  );
}

// Grows from zero width so the controls after it slide over instead of jumping
function Reveal({
  open,
  className,
  children,
}: {
  open: boolean;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      inert={!open}
      className={cn(
        "grid transition-[grid-template-columns] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)] motion-reduce:transition-none",
        open ? "grid-cols-[1fr]" : "grid-cols-[0fr]",
      )}
    >
      <div
        className={cn(
          "flex min-w-0 overflow-hidden transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.23,1,0.32,1)] motion-reduce:transition-none",
          open ? "translate-x-0 opacity-100" : "-translate-x-1 opacity-0",
          className,
        )}
      >
        {children}
      </div>
    </div>
  );
}

function CopyLinesButton({ label, text }: { label: string; text: () => string }) {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) {
      return;
    }
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [copied]);

  return (
    <Button
      variant="outline"
      size="sm"
      title={`Copy ${label}`}
      className="h-7 min-w-0 gap-2 px-2 font-normal text-gray-11 tabular-nums [&_svg]:size-3"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text());
          toast.success("Copied to clipboard");
          setCopied(true);
        } catch (error) {
          toast.error("Failed to copy build logs", {
            description: error instanceof Error ? error.message : undefined,
          });
        }
      }}
    >
      {copied ? <IconCheckOutline12 /> : <IconCloneOutline12 />}
      <span className="truncate">{label}</span>
    </Button>
  );
}
