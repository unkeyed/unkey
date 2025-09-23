import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useEffect, useMemo, useRef, useState } from "react";

type LogEntry = {
  type: "build" | "gateway";
  id: string;
  timestamp: number;
  message: string;
  level?: "warning" | "error";
};

type LogFilter = "all" | "warnings" | "errors";

type UseDeploymentLogsProps = {
  deploymentId: string;
  showBuildSteps: boolean;
};

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
  scrollRef: React.RefObject<HTMLDivElement>;
};

export function useDeploymentLogs({
  deploymentId,
  showBuildSteps,
}: UseDeploymentLogsProps): UseDeploymentLogsReturn {
  const [logFilter, setLogFilter] = useState<LogFilter>("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [isExpanded, setIsExpanded] = useState(false);
  const [showFade, setShowFade] = useState(true);
  const [storedLogs, setStoredLogs] = useState<Map<string, LogEntry>>(new Map());
  const scrollRef = useRef<HTMLDivElement>(null);
  const { queryTime: timestamp } = useQueryTime();

  const { data: buildData, isLoading: buildLoading } = trpc.deploy.deployment.buildSteps.useQuery(
    { deploymentId },
    { enabled: showBuildSteps && isExpanded, refetchInterval: 500 },
  );

  const { data: gatewayData, isLoading: gatewayLoading } = trpc.logs.queryLogs.useQuery(
    {
      limit: 20,
      endTime: timestamp,
      startTime: timestamp,
      // TODO: Exclude some hosts
      host: { filters: [] },
      method: { filters: [] },
      path: { filters: [] },
      status: { filters: [] },
      requestId: null,
      since: "1m",
    },
    {
      enabled: !showBuildSteps && isExpanded,
      refetchInterval: 2000,
      refetchOnWindowFocus: false,
    },
  );

  // Update stored logs when data changes
  useEffect(() => {
    if (showBuildSteps && buildData?.logs) {
      const logMap = new Map<string, LogEntry>();
      buildData.logs.forEach((log) => {
        logMap.set(log.id, {
          type: "build",
          id: log.id,
          timestamp: log.timestamp,
          message: log.message,
        });
      });
      setStoredLogs(logMap);
    }
  }, [showBuildSteps, buildData]);

  useEffect(() => {
    if (!showBuildSteps && gatewayData?.logs) {
      setStoredLogs((prev) => {
        const newMap = new Map(prev);

        gatewayData.logs.forEach((log) => {
          let level: "warning" | "error" | undefined;
          if (log.error || log.response_status >= 500) {
            level = "error";
          } else if (log.response_status >= 400) {
            level = "warning";
          }

          newMap.set(log.request_id, {
            type: "gateway",
            id: log.request_id,
            timestamp: log.time,
            message: `${log.method} ${log.path} → ${log.response_status} (${log.service_latency}ms)`,
            level,
          });
        });

        const sortedEntries = Array.from(newMap.entries())
          .sort((a, b) => a[1].timestamp - b[1].timestamp)
          .slice(0, 200);

        return new Map(sortedEntries);
      });
    }
  }, [showBuildSteps, gatewayData]);

  const logs = useMemo(() => {
    return Array.from(storedLogs.values()).sort((a, b) => a.timestamp - b.timestamp);
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
      filtered = logs.filter((log) => log.type === "build" || log.level === "warning");
    } else if (logFilter === "errors") {
      filtered = logs.filter((log) => log.type === "build" || log.level === "error");
    }

    if (searchTerm.trim()) {
      filtered = filtered.filter((log) =>
        log.message.toLowerCase().includes(searchTerm.toLowerCase()),
      );
    }

    return filtered;
  }, [logs, logFilter, searchTerm]);

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
      setTimeout(resetScroll, 50);
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
    isLoading: showBuildSteps ? buildLoading : gatewayLoading,
    setLogFilter,
    setSearchTerm,
    setExpanded,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  };
}
