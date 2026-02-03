"use client";
import { extractResponseField, safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import type { Log } from "@unkey/clickhouse/src/logs";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";
import { type ReactNode, createContext, useContext, useEffect, useMemo, useState } from "react";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import { LogMetaSection } from "./components/log-meta";
import { LogSection } from "./components/log-section";

export const DEFAULT_DRAGGABLE_WIDTH = 500;
export const EMPTY_TEXT = "<EMPTY>";

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

export type StandardLogTypes = Log | RatelimitLog;
export type SupportedLogTypes = StandardLogTypes | KeysOverviewLog | AuditLog | RuntimeLog;

type LogDetailsContextValue = {
  animated: boolean;
  isOpen: boolean;
  log: SupportedLogTypes;
};

const LogDetailsContext = createContext<LogDetailsContextValue>({
  animated: false,
  isOpen: true,
  log: {} as SupportedLogTypes,
});

const useLogDetailsContext = () => useContext(LogDetailsContext);

// Helper functions for standard logs
const createLogSections = (log: Log | RatelimitLog) => [
  {
    title: "Request Header",
    content: log.request_headers.length ? log.request_headers : EMPTY_TEXT,
  },
  {
    title: "Request Body",
    content:
      JSON.stringify(safeParseJson(log.request_body), null, 2) === "null" ? (
        <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>
      ) : (
        JSON.stringify(safeParseJson(log.request_body), null, 2)
      ),
  },
  {
    title: "Response Header",
    content: log.response_headers.length ? log.response_headers : EMPTY_TEXT,
  },
  {
    title: "Response Body",
    content:
      JSON.stringify(safeParseJson(log.response_body), null, 2) === "null" ? (
        <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>
      ) : (
        JSON.stringify(safeParseJson(log.response_body), null, 2)
      ),
  },
];

const createMetaContent = (log: SupportedLogTypes) => {
  // Handle KeysOverviewLog meta differently
  if ("key_details" in log && (log.key_details as { meta: string })?.meta) {
    try {
      const parsedMeta = JSON.parse((log.key_details as { meta: string })?.meta);
      return JSON.stringify(parsedMeta, null, 2);
    } catch {
      return <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>;
    }
  }

  // Standard log meta handling
  if ("request_body" in log || "response_body" in log) {
    const meta = extractResponseField(log as Log | RatelimitLog, "meta");
    return JSON.stringify(meta, null, 2) === "null" ? (
      <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>
    ) : (
      JSON.stringify(meta, null, 2)
    );
  }

  return <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>;
};

// Type guards
const isStandardLog = (log: SupportedLogTypes): log is Log | RatelimitLog => {
  return "request_headers" in log && "response_headers" in log;
};

// const isRuntimeLog = (log: SupportedLogTypes): log is RuntimeLog => {
//   return "deployment_id" in log
// };

// Main LogDetails component
type LogDetailsProps = {
  distanceToTop: number;
  log: SupportedLogTypes | null;
  onClose: () => void;
  animated?: boolean;
  children: ReactNode;
};

export const LogDetails = ({
  distanceToTop,
  log,
  onClose,
  animated = false,
  children,
}: LogDetailsProps) => {
  const [isOpen, setIsOpen] = useState(false);

  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);

  useEffect(() => {
    if (!animated) {
      return;
    }

    if (log) {
      const timer = setTimeout(() => setIsOpen(true), 50);
      return () => clearTimeout(timer);
    }
    setIsOpen(false);
  }, [log, animated]);

  useEffect(() => {
    if (!animated) {
      setIsOpen(Boolean(log));
    }
  }, [log, animated]);

  if (!log) {
    return null;
  }

  const handleClose = () => {
    if (animated) {
      setIsOpen(false);
      setTimeout(onClose, 300);
    } else {
      onClose();
    }
  };

  const baseClasses = "bg-gray-1 font-mono drop-shadow-2xl z-20";
  const animationClasses = animated
    ? cn(
        "transition-all duration-300 ease-out",
        isOpen ? "translate-x-0 opacity-100" : "translate-x-full opacity-0",
      )
    : "";
  const staticClasses = animated ? "" : "absolute right-0 overflow-y-auto";

  return (
    <ResizablePanel
      onClose={handleClose}
      className={cn(baseClasses, animationClasses, staticClasses)}
      style={{
        ...panelStyle,
        width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
        ...(animated && {
          willChange: isOpen ? "transform, opacity" : "auto",
        }),
      }}
    >
      <div className={animated ? "h-full overflow-y-auto p-4" : ""}>
        <LogDetailsContext.Provider value={{ animated, isOpen, log }}>
          {children}
        </LogDetailsContext.Provider>
      </div>
    </ResizablePanel>
  );
};

// Section wrapper with animation
type SectionProps = {
  children: ReactNode;
  delay?: number;
  translateX?: "translate-x-6" | "translate-x-8";
};

const Section = ({ children, delay = 0, translateX = "translate-x-8" }: SectionProps) => {
  const { animated, isOpen } = useLogDetailsContext();

  if (!animated) {
    return <>{children}</>;
  }

  return (
    <div
      className={cn(
        "transition-all duration-300 ease-out",
        isOpen ? "translate-x-0 opacity-100" : `${translateX} opacity-0`,
      )}
      style={{ transitionDelay: isOpen ? `${delay}ms` : "0ms" }}
    >
      {children}
    </div>
  );
};

// Standard log sections (only works for standard logs)
const Sections = ({
  startDelay = 150,
  staggerDelay = 50,
}: {
  startDelay?: number;
  staggerDelay?: number;
}) => {
  const { log } = useLogDetailsContext();

  if (!isStandardLog(log)) {
    console.warn("LogDetails.Sections can only be used with standard logs (Log | RatelimitLog)");
    return null;
  }

  const sections = createLogSections(log);

  return (
    <>
      {sections.map((section, index) => (
        <Section key={section.title} delay={startDelay + index * staggerDelay}>
          <LogSection details={section.content} title={section.title} />
        </Section>
      ))}
    </>
  );
};

// Custom sections wrapper for flexible content
type CustomSectionsProps = {
  children: ReactNode;
  startDelay?: number;
  staggerDelay?: number;
};

const CustomSections = ({ children, startDelay = 150, staggerDelay = 50 }: CustomSectionsProps) => {
  const childArray = Array.isArray(children) ? children : [children];

  return (
    <>
      {childArray.map((child, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: its fine
        <Section key={index} delay={startDelay + index * staggerDelay}>
          {child}
        </Section>
      ))}
    </>
  );
};

// Spacer with animation
const Spacer = ({ delay = 0 }: { delay?: number }) => {
  const { animated, isOpen } = useLogDetailsContext();

  return (
    <div
      className={
        animated
          ? cn(
              "mt-3 transition-all duration-300 ease-out",
              isOpen ? "translate-x-0 opacity-100" : "translate-x-8 opacity-0",
            )
          : "mt-3"
      }
      style={animated ? { transitionDelay: isOpen ? `${delay}ms` : "0ms" } : undefined}
    />
  );
};

// Meta section
const Meta = ({ delay = 400 }: { delay?: number }) => {
  const { log } = useLogDetailsContext();
  const content = createMetaContent(log);

  return (
    <Section delay={delay}>
      <LogMetaSection content={content} />
    </Section>
  );
};

// Generic Header wrapper - allows any header component or falls back to default
const Header = ({
  delay = 100,
  translateX = "translate-x-6" as const,
  onClose,
  children,
}: {
  delay?: number;
  translateX?: "translate-x-6" | "translate-x-8";
  onClose?: () => void;
  children?: ReactNode;
}) => {
  const { log } = useLogDetailsContext();

  return (
    <Section delay={delay} translateX={translateX}>
      {children ||
        (onClose &&
          (isStandardLog(log) ? (
            <LogHeader log={log as StandardLogTypes} onClose={onClose} />
          ) : null))}
    </Section>
  );
};

// Generic Footer wrapper - allows any footer component or falls back to default
const Footer = ({
  delay = 375,
  children,
}: {
  delay?: number;
  children?: ReactNode;
}) => {
  const { log } = useLogDetailsContext();

  return (
    <Section delay={delay}>
      {children ||
        (isStandardLog(log) ? (
          <div className="px-4">
            <LogFooter log={log} />{" "}
          </div>
        ) : null)}
    </Section>
  );
};

// Compound components
LogDetails.Section = Section;
LogDetails.Sections = Sections;
LogDetails.CustomSections = CustomSections;
LogDetails.Spacer = Spacer;
LogDetails.Meta = Meta;
LogDetails.Header = Header;
LogDetails.Footer = Footer;
LogDetails.useContext = useLogDetailsContext;
LogDetails.createMetaContent = createMetaContent;
