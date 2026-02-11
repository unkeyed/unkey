import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo, useRef, useState } from "react";

const GATEWAY_LOGS_REFETCH_INTERVAL = 5000;
const GATEWAY_LOGS_LIMIT = 50;
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

type UseDeploymentLogsProps = {
  deploymentId: string | null;
  projectId: string;
};

type UseDeploymentLogsReturn = {
  logFilter: LogFilter;
  searchTerm: string;
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
  handleScroll: (e: React.UIEvent<HTMLDivElement>) => void;
  handleFilterChange: (filter: LogFilter) => void;
  handleSearchChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  scrollRef: React.MutableRefObject<HTMLDivElement>;
};

export function useDeploymentLogs({
  deploymentId,
  projectId,
}: UseDeploymentLogsProps): UseDeploymentLogsReturn {
  const [logFilter, setLogFilter] = useState<LogFilter>("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [showFade, setShowFade] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null) as React.MutableRefObject<HTMLDivElement>;

  const deployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) => eq(deployment.projectId, projectId))
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [projectId, deploymentId],
  );
  const environmentId = deployment.data.at(0)?.environmentId ?? "";
  const { data: sentinelData, isLoading: sentinelLoading } =
    trpc.deploy.sentinelLogs.query.useQuery(
      {
        projectId,
        deploymentId: deploymentId ?? "",
        environmentId,
        limit: GATEWAY_LOGS_LIMIT,
        since: "6h",
      },
      {
        enabled: Boolean(deploymentId) && Boolean(environmentId),
        refetchInterval: GATEWAY_LOGS_REFETCH_INTERVAL,
        refetchOnWindowFocus: false,
      },
    );

  const logs = useMemo(() => {
    if (!sentinelData?.logs) {
      return [];
    }

    return sentinelData.logs.map((log) => {
      let level: "warning" | "error" | undefined;
      if (log.response_status >= ERROR_STATUS_THRESHOLD) {
        level = "error";
      } else if (log.response_status >= WARNING_STATUS_THRESHOLD) {
        level = "warning";
      }

      return {
        type: "sentinel" as const,
        id: log.request_id,
        timestamp: log.time,
        message: `${log.response_status} ${log.method} ${log.path} (${log.total_latency}ms)`,
        level,
      };
    });
  }, [sentinelData]);

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
    isLoading: sentinelLoading,
    setLogFilter,
    setSearchTerm,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  };
}
