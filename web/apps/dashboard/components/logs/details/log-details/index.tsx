"use client";
import { extractResponseField, safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import type { EnrichedRatelimitLog } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/components/table/hooks/use-logs-query";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import type { Log } from "@unkey/clickhouse/src/logs";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { type ReactNode, createContext, useContext, useMemo } from "react";
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

export type StandardLogTypes = Log | EnrichedRatelimitLog;
export type SupportedLogTypes =
  | StandardLogTypes
  | KeysOverviewLog
  | AuditLog
  | RuntimeLog
  | SentinelLogsResponse;

type LogDetailsContextValue = {
  log: SupportedLogTypes;
  onClose: () => void;
};

const LogDetailsContext = createContext<LogDetailsContextValue>({
  log: {} as SupportedLogTypes,
  onClose: () => {},
});

const useLogDetailsContext = () => useContext(LogDetailsContext);

const createLogSections = (log: Log | EnrichedRatelimitLog) => [
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
  if ("key_details" in log && (log.key_details as { meta: string })?.meta) {
    try {
      const parsedMeta = JSON.parse((log.key_details as { meta: string })?.meta);
      return JSON.stringify(parsedMeta, null, 2);
    } catch {
      return <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>;
    }
  }

  if (isStandardLog(log)) {
    const meta = extractResponseField(log, "meta");
    return JSON.stringify(meta, null, 2) === "null" ? (
      <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>
    ) : (
      JSON.stringify(meta, null, 2)
    );
  }

  return <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>;
};

const isStandardLog = (log: SupportedLogTypes): log is Log | EnrichedRatelimitLog => {
  return "request_headers" in log && "response_headers" in log;
};

type LogDetailsProps = {
  distanceToTop: number;
  log: SupportedLogTypes | null;
  onClose: () => void;
  children: ReactNode;
};

export const LogDetails = ({ distanceToTop, log, onClose, children }: LogDetailsProps) => {
  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);

  if (!log) {
    return null;
  }

  return (
    <ResizablePanel
      onClose={onClose}
      className="bg-gray-1 font-mono drop-shadow-2xl z-20 absolute right-0 overflow-y-auto"
      style={{
        ...panelStyle,
        width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
      }}
    >
      <LogDetailsContext.Provider value={{ log, onClose }}>{children}</LogDetailsContext.Provider>
    </ResizablePanel>
  );
};

const Sections = () => {
  const { log } = useLogDetailsContext();

  if (!isStandardLog(log)) {
    console.warn(
      "LogDetails.Sections can only be used with standard logs (Log | EnrichedRatelimitLog)",
    );
    return null;
  }

  const sections = createLogSections(log);

  return (
    <>
      {sections.map((section) => (
        <LogSection key={section.title} details={section.content} title={section.title} />
      ))}
    </>
  );
};

const CustomSections = ({ children }: { children: ReactNode }) => {
  return <>{children}</>;
};

const Spacer = () => {
  return <div className="mt-3" />;
};

const Meta = () => {
  const { log } = useLogDetailsContext();
  const content = createMetaContent(log);

  return <LogMetaSection content={content} />;
};

const Header = ({
  onClose,
  children,
}: {
  onClose?: () => void;
  children?: ReactNode;
}) => {
  const { log } = useLogDetailsContext();

  return (
    <>
      {children ||
        (onClose &&
          (isStandardLog(log) ? (
            <LogHeader log={log as StandardLogTypes} onClose={onClose} />
          ) : null))}
    </>
  );
};

const Footer = ({
  children,
}: {
  children?: ReactNode;
}) => {
  const { log } = useLogDetailsContext();

  return (
    <>
      {children ||
        (isStandardLog(log) ? (
          <div className="px-4">
            <LogFooter log={log} />{" "}
          </div>
        ) : null)}
    </>
  );
};

const Section = ({ children }: { children: ReactNode }) => {
  return <>{children}</>;
};

LogDetails.Section = Section;
LogDetails.Sections = Sections;
LogDetails.CustomSections = CustomSections;
LogDetails.Spacer = Spacer;
LogDetails.Meta = Meta;
LogDetails.Header = Header;
LogDetails.Footer = Footer;
LogDetails.useContext = useLogDetailsContext;
LogDetails.createMetaContent = createMetaContent;
