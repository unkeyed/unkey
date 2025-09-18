"use client";
import { extractResponseField, safeParseJson } from "@/app/(app)/logs/utils";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";
import { type ReactNode, createContext, useContext, useEffect, useMemo, useState } from "react";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import { LogMetaSection } from "./components/log-meta";
import { LogSection } from "./components/log-section";

export const DEFAULT_DRAGGABLE_WIDTH = 500;
const EMPTY_TEXT = "<EMPTY>";

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

export type SupportedLogTypes = Log | RatelimitLog;

const LogDetailsContext = createContext<{
  animated: boolean;
  isOpen: boolean;
  log: SupportedLogTypes;
}>({ animated: false, isOpen: true, log: {} as SupportedLogTypes });

const useLogDetailsContext = () => useContext(LogDetailsContext);

// Helper functions
const createLogSections = (log: SupportedLogTypes) => [
  {
    title: "Request Header",
    content: log.request_headers.length ? log.request_headers : EMPTY_TEXT,
  },
  {
    title: "Request Body",
    content:
      JSON.stringify(safeParseJson(log.request_body), null, 2) === "null"
        ? EMPTY_TEXT
        : JSON.stringify(safeParseJson(log.request_body), null, 2),
  },
  {
    title: "Response Header",
    content: log.response_headers.length ? log.response_headers : EMPTY_TEXT,
  },
  {
    title: "Response Body",
    content:
      JSON.stringify(safeParseJson(log.response_body), null, 2) === "null"
        ? EMPTY_TEXT
        : JSON.stringify(safeParseJson(log.response_body), null, 2),
  },
];

const createMetaContent = (log: SupportedLogTypes) => {
  const meta = extractResponseField(log, "meta");
  return JSON.stringify(meta, null, 2) === "null" ? EMPTY_TEXT : JSON.stringify(meta, null, 2);
};

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
  const staticClasses = animated ? "" : "absolute right-0 overflow-y-auto p-4";

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

// Standard log sections
const Sections = ({
  startDelay = 150,
  staggerDelay = 50,
}: {
  startDelay?: number;
  staggerDelay?: number;
}) => {
  const { log } = useLogDetailsContext();
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

// Header compound component
const Header = ({
  delay = 100,
  onClose,
}: {
  delay?: number;
  onClose: () => void;
}) => {
  const { log } = useLogDetailsContext();

  return (
    <Section delay={delay} translateX="translate-x-6">
      <LogHeader log={log} onClose={onClose} />
    </Section>
  );
};

// Footer compound component
const Footer = ({ delay = 375 }: { delay?: number }) => {
  const { log } = useLogDetailsContext();

  return (
    <Section delay={delay}>
      <LogFooter log={log} />
    </Section>
  );
};

// Compound components
LogDetails.Section = Section;
LogDetails.Sections = Sections;
LogDetails.Spacer = Spacer;
LogDetails.Meta = Meta;
LogDetails.Header = Header;
LogDetails.Footer = Footer;
LogDetails.useContext = useLogDetailsContext;
LogDetails.createMetaContent = createMetaContent;
