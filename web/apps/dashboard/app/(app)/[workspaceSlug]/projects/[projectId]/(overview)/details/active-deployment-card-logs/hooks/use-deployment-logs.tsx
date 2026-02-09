import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useEffect, useMemo, useRef, useState } from "react";
import { EXCLUDED_HOSTS } from "../../../sentinel-logs/constants";

const GATEWAY_LOGS_REFETCH_INTERVAL = 2000;
const GATEWAY_LOGS_LIMIT = 20;
const GATEWAY_LOGS_SINCE = "1m";
const MAX_STORED_LOGS = 200;
const SCROLL_RESET_DELAY = 50;
const ERROR_STATUS_THRESHOLD = 500;
const WARNING_STATUS_THRESHOLD = 400;

type LogEntry = {
  type: "sentinel";
  id: string;
  timestamp: number;
  message: string;
  level?: "warning" | "error";
};

type LogFilter = "all" | "warnings" | "errors";

type UseDeploymentLogsReturn = {
  logFilter: LogFilter;
  searchTerm: string;
  isExpanded: boolean;
  showFade: boolean;
  filteredLogs: LogEntry[];
  logCounts: {
    total: number;
    warnings: number;
    errors: number;
  };
  isLoading: boolean;
  setLogFilter: (filter: LogFilter) => void;
  setSearchTerm: (term: string) => void;
  setExpanded: (expanded: boolean) => void;
  handleScroll: (e: React.UIEvent<HTMLDivElement>) => void;
  handleFilterChange: (filter: LogFilter) => void;
  handleSearchChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  scrollRef: React.MutableRefObject<HTMLDivElement>;
};

export function useDeploymentLogs(): UseDeploymentLogsReturn {
  const [logFilter, setLogFilter] = useState<LogFilter>("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [isExpanded, setIsExpanded] = useState(false);
  const [showFade, setShowFade] = useState(true);
  const [storedLogs, setStoredLogs] = useState<Map<string, LogEntry>>(new Map());
  const scrollRef = useRef<HTMLDivElement>(null) as React.MutableRefObject<HTMLDivElement>;
  const { queryTime: timestamp } = useQueryTime();

  const { data: sentinelData, isLoading: sentinelLoading } = trpc.logs.queryLogs.useQuery(
    {
      limit: GATEWAY_LOGS_LIMIT,
      endTime: timestamp,
      startTime: timestamp,
      host: { filters: [], exclude: EXCLUDED_HOSTS },
      method: { filters: [] },
      path: { filters: [] },
      status: { filters: [] },
      requestId: null,
      since: GATEWAY_LOGS_SINCE,
    },
    {
      enabled: isExpanded,
      refetchInterval: GATEWAY_LOGS_REFETCH_INTERVAL,
      refetchOnWindowFocus: false,
    },
  );

  // Update stored logs when sentinel data changes
  useEffect(() => {
    if (sentinelData?.logs) {
      setStoredLogs((prev) => {
        const newMap = new Map(prev);

        sentinelData.logs.forEach((log) => {
          let level: "warning" | "error" | undefined;
          if (log.response_status >= ERROR_STATUS_THRESHOLD) {
            level = "error";
          } else if (log.response_status >= WARNING_STATUS_THRESHOLD) {
            level = "warning";
          }

          newMap.set(log.request_id, {
            type: "sentinel",
            id: log.request_id,
            timestamp: log.time,
            message: `${log.response_status} ${log.method} ${log.path} (${log.service_latency}ms)`,
            level,
          });
        });

        const sortedEntries = Array.from(newMap.entries())
          .sort((a, b) => b[1].timestamp - a[1].timestamp)
          .slice(0, MAX_STORED_LOGS);

        return new Map(sortedEntries);
      });
    }
  }, [sentinelData]);

  const logs = useMemo(() => {
    return Array.from(storedLogs.values()).sort((a, b) => b.timestamp - a.timestamp);
  }, [storedLogs]);

  const logCounts = useMemo(() => {
    const warnings = logs.filter((log) => log.level === "warning").length;
    const errors = logs.filter((log) => log.level === "error").length;

    return {
      total: logs.length,
      warnings,
      errors,
    };
  }, [logs]);

  const filteredLogs = useMemo(() => {
    let filtered = logs;

    if (logFilter === "warnings") {
      filtered = logs.filter((log) => log.level === "warning");
    } else if (logFilter === "errors") {
      filtered = logs.filter((log) => log.level === "error");
    }

    if (searchTerm.trim()) {
      filtered = filtered.filter((log) =>
        log.message.toLowerCase().includes(searchTerm.toLowerCase()),
      );
    }

    return filtered;
  }, [logs, logFilter, searchTerm]);

  // Auto-expand when logs are available
  useEffect(() => {
    if (logs.length > 0) {
      setIsExpanded(true);
    }
  }, [logs.length]);

  const resetScroll = () => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
      setShowFade(true);
    }
  };

  const setExpanded = (expanded: boolean) => {
    setIsExpanded(expanded);
    if (!expanded) {
      setTimeout(resetScroll, SCROLL_RESET_DELAY);
    }
  };

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1;
    setShowFade(!isAtBottom);
  };

  const handleFilterChange = (filter: LogFilter) => {
    setLogFilter(filter);
    resetScroll();
  };

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value);
    resetScroll();
  };

  return {
    logFilter,
    searchTerm,
    isExpanded,
    showFade,
    filteredLogs,
    logCounts,
    isLoading: sentinelLoading,
    setLogFilter,
    setSearchTerm,
    setExpanded,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  };
}
