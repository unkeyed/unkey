import { format } from "date-fns";
import { useMemo, useRef, useState } from "react";

type LogEntry = {
  timestamp: string;
  level?: "info" | "warning" | "error";
  message: string;
};

type LogFilter = "all" | "errors" | "warnings";

type UseDeploymentLogsProps = {
  deploymentId: string;
};

type UseDeploymentLogsReturn = {
  // State
  logFilter: LogFilter;
  searchTerm: string;
  isExpanded: boolean;
  showFade: boolean;
  // Computed
  filteredLogs: LogEntry[];
  logCounts: {
    total: number;
    errors: number;
    warnings: number;
  };
  // Loading state
  isLoading: boolean;
  // Actions
  setLogFilter: (filter: LogFilter) => void;
  setSearchTerm: (term: string) => void;
  setExpanded: (expanded: boolean) => void;
  handleScroll: (e: React.UIEvent<HTMLDivElement>) => void;
  handleFilterChange: (filter: LogFilter) => void;
  handleSearchChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  // Refs
  scrollRef: React.RefObject<HTMLDivElement>;
};

export function useDeploymentLogs({
  deploymentId,
}: UseDeploymentLogsProps): UseDeploymentLogsReturn {
  console.debug(deploymentId);
  const [logFilter, setLogFilter] = useState<LogFilter>("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [isExpanded, setIsExpanded] = useState(false);
  const [showFade, setShowFade] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Fetch logs via tRPC
  const isLoading = false;
  const logsData = { logs: [] as any[] };

  // Transform tRPC logs to match the expected format
  const logs = useMemo((): LogEntry[] => {
    if (!logsData?.logs) {
      return [];
    }

    return logsData.logs.map((log) => ({
      timestamp: format(new Date(log.timestamp), "HH:mm:ss.SSS"),
      level: log.level,
      message: log.message,
    }));
  }, [logsData]);

  // Calculate log counts
  const logCounts = useMemo(
    () => ({
      total: logs.length,
      errors: logs.filter((log) => log.level === "error").length,
      warnings: logs.filter((log) => log.level === "warning").length,
    }),
    [logs],
  );

  // Filter logs by level and search term
  const filteredLogs = useMemo(() => {
    let filtered = logs;

    // Apply level filter
    if (logFilter === "errors") {
      filtered = logs.filter((log) => log.level === "error");
    } else if (logFilter === "warnings") {
      filtered = logs.filter((log) => log.level === "warning");
    }

    // Apply search filter
    if (searchTerm.trim()) {
      filtered = filtered.filter(
        (log) =>
          log.message.toLowerCase().includes(searchTerm.toLowerCase()) ||
          log.timestamp.includes(searchTerm) ||
          log.level?.toLowerCase().includes(searchTerm.toLowerCase()),
      );
    }

    return filtered;
  }, [logs, logFilter, searchTerm]);

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
    // State
    logFilter,
    searchTerm,
    isExpanded,
    showFade,
    // Computed
    filteredLogs,
    logCounts,
    // Loading state
    isLoading,
    // Actions
    setLogFilter,
    setSearchTerm,
    setExpanded,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    // Refs
    scrollRef,
  };
}
