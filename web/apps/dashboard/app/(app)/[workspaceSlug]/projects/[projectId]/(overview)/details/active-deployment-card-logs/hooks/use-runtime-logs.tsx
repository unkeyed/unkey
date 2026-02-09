import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useEffect, useMemo, useRef, useState } from "react";

const RUNTIME_LOGS_REFETCH_INTERVAL = 5000;
const RUNTIME_LOGS_LIMIT = 50;
const RUNTIME_LOGS_SINCE = "6h";
const MAX_STORED_LOGS = 200;

type LogFilter = "all" | "warnings" | "errors";

type UseRuntimeLogsProps = {
  projectId: string;
  deploymentId: string;
};

type UseRuntimeLogsReturn = {
  logFilter: LogFilter;
  searchTerm: string;
  showFade: boolean;
  filteredLogs: RuntimeLog[];
  logCounts: {
    total: number;
    warnings: number;
    errors: number;
  };
  isLoading: boolean;
  handleScroll: (e: React.UIEvent<HTMLDivElement>) => void;
  handleFilterChange: (filter: LogFilter) => void;
  handleSearchChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  scrollRef: React.MutableRefObject<HTMLDivElement>;
};

export function useRuntimeLogs({
  projectId,
  deploymentId,
}: UseRuntimeLogsProps): UseRuntimeLogsReturn {
  const [logFilter, setLogFilter] = useState<LogFilter>("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [showFade, setShowFade] = useState(true);
  const [storedLogs, setStoredLogs] = useState<Map<string, RuntimeLog>>(new Map());
  const scrollRef = useRef<HTMLDivElement>(null) as React.MutableRefObject<HTMLDivElement>;
  const { queryTime: timestamp } = useQueryTime();

  const { data, isLoading } = trpc.deploy.runtimeLogs.query.useQuery(
    {
      projectId,
      deploymentId,
      limit: RUNTIME_LOGS_LIMIT,
      startTime: timestamp,
      endTime: timestamp,
      since: RUNTIME_LOGS_SINCE,
      severity: null,
      message: null,
    },
    {
      refetchInterval: RUNTIME_LOGS_REFETCH_INTERVAL,
      refetchOnWindowFocus: false,
    },
  );

  useEffect(() => {
    if (data?.logs) {
      setStoredLogs((prev) => {
        const newMap = new Map(prev);

        data.logs.forEach((log) => {
          newMap.set(`${log.deployment_id}-${log.time}`, log);
        });

        const sortedEntries = Array.from(newMap.entries())
          .sort((a, b) => b[1].time - a[1].time)
          .slice(0, MAX_STORED_LOGS);

        return new Map(sortedEntries);
      });
    }
  }, [data]);

  const logs = useMemo(() => {
    return Array.from(storedLogs.values()).sort((a, b) => b.time - a.time);
  }, [storedLogs]);

  const logCounts = useMemo(() => {
    const warnings = logs.filter((log) => log.severity.toUpperCase() === "WARN").length;
    const errors = logs.filter((log) => log.severity.toUpperCase() === "ERROR").length;

    return {
      total: logs.length,
      warnings,
      errors,
    };
  }, [logs]);

  const filteredLogs = useMemo(() => {
    let filtered = logs;

    if (logFilter === "warnings") {
      filtered = logs.filter((log) => log.severity.toUpperCase() === "WARN");
    } else if (logFilter === "errors") {
      filtered = logs.filter((log) => log.severity.toUpperCase() === "ERROR");
    }

    if (searchTerm.trim()) {
      filtered = filtered.filter((log) =>
        log.message.toLowerCase().includes(searchTerm.toLowerCase()),
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
    showFade,
    filteredLogs,
    logCounts,
    isLoading,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  };
}
