"use client";
import {
  ChevronDown,
  CircleInfo,
  CircleWarning,
  CircleXMark,
  InputSearch,
  TriangleWarning,
} from "@unkey/icons";
import type { ChangelogEntry } from "@unkey/proto";
import {
  Badge,
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type React from "react";
import { useMemo, useState } from "react";

type DiffViewerContentProps = {
  changelog: ChangelogEntry[];
  fromDeployment?: string;
  toDeployment?: string;
};

export const DiffViewerContent: React.FC<DiffViewerContentProps> = ({
  changelog,
  fromDeployment = "Previous",
  toDeployment = "Current",
}) => {
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [filters, setFilters] = useState({
    level: null as number | null,
    operation: "all",
    searchQuery: "",
  });

  const stats = useMemo(() => {
    const breaking = changelog.filter((c) => c.level === 3).length;
    const warning = changelog.filter((c) => c.level === 2).length;
    const info = changelog.filter((c) => c.level === 1).length;
    const total = changelog.length;
    const operations = [...new Set(changelog.map((c) => c.operation))];
    const paths = [...new Set(changelog.map((c) => c.path))];
    return { breaking, warning, info, total, operations, paths };
  }, [changelog]);

  const filteredChanges = useMemo(() => {
    return changelog.filter((change) => {
      if (filters.level !== null && change.level !== filters.level) {
        return false;
      }
      if (filters.operation !== "all" && change.operation !== filters.operation) {
        return false;
      }
      if (filters.searchQuery) {
        const query = filters.searchQuery.toLowerCase();
        return (
          change.text.toLowerCase().includes(query) ||
          change.path.toLowerCase().includes(query) ||
          change.id.toLowerCase().includes(query)
        );
      }
      return true;
    });
  }, [changelog, filters]);

  const groupedChanges = useMemo(() => {
    const grouped: Record<string, Record<string, ChangelogEntry[]>> = {};
    filteredChanges.forEach((change) => {
      if (!grouped[change.path]) {
        grouped[change.path] = {};
      }
      if (!grouped[change.path][change.operation]) {
        grouped[change.path][change.operation] = [];
      }
      grouped[change.path][change.operation].push(change);
    });
    return grouped;
  }, [filteredChanges]);

  const togglePathExpansion = (path: string) => {
    const newExpanded = new Set(expandedPaths);
    newExpanded.has(path) ? newExpanded.delete(path) : newExpanded.add(path);
    setExpandedPaths(newExpanded);
  };

  const getSeverityIcon = (level: number) => {
    if (level === 3) {
      return <TriangleWarning size="sm-regular" className="text-errorA-11" />;
    }
    if (level === 2) {
      return <CircleWarning size="sm-regular" className="text-warningA-11" />;
    }
    return <CircleInfo size="sm-regular" className="text-grayA-9" />;
  };

  const getSeverityColor = (level: number) => {
    if (level === 3) {
      return "border border-error-6 bg-errorA-2";
    }
    if (level === 2) {
      return "border border-warning-6 bg-warningA-2";
    }
    return "border border-gray-4 bg-grayA-1";
  };

  if (!changelog || changelog.length === 0) {
    return (
      <div className="flex flex-col items-center gap-4 px-8 py-12 text-center">
        <div className="relative">
          <div className="absolute inset-0 bg-gradient-to-r from-accent-4 to-accent-3 rounded-full blur-xl opacity-20 transition-opacity duration-300 animate-pulse" />
          <div className="relative bg-gray-3 rounded-full p-3 transition-all duration-200">
            <CircleInfo
              className="text-grayA-9 size-6 transition-all duration-200 animate-pulse"
              style={{ animationDuration: "2s" }}
            />
          </div>
        </div>
        <div className="space-y-1">
          <h3 className="text-grayA-12 font-medium text-sm">No noteworthy changes</h3>
          <p className="text-grayA-9 text-xs max-w-[280px] leading-relaxed">
            The specifications for <span className="text-grayA-11">{fromDeployment} </span>
            and <span className="text-grayA-11">{toDeployment} </span>
            are functionally identical.
          </p>
        </div>
      </div>
    );
  }
  return (
    <>
      {/* Stats header */}
      <div className="px-3 pb-3">
        <div className="flex justify-between items-center">
          <div className="flex flex-col gap-1">
            <div className="text-accent-12 font-medium text-[13px]">API Changes</div>
            <div className="text-grayA-9 text-xs">
              {stats.total} changes • {stats.paths.length} endpoints
            </div>
          </div>
          <div className="flex items-center gap-2">
            {stats.breaking > 0 && (
              <Badge variant="error" className="gap-1.5">
                <TriangleWarning size="sm-regular" className="shrink-0" />
                <span className="text-xs font-medium">{stats.breaking} breaking</span>
              </Badge>
            )}
            {stats.warning > 0 && (
              <Badge variant="warning" className="gap-1.5">
                <CircleWarning size="sm-regular" className="shrink-0" />
                <span className="text-xs">
                  {stats.warning} warning{stats.warning !== 1 ? "s" : ""}
                </span>
              </Badge>
            )}
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="px-3 pb-2 flex gap-2.5 items-center">
        <Input
          type="text"
          value={filters.searchQuery}
          onChange={(e) => setFilters((p) => ({ ...p, searchQuery: e.target.value }))}
          placeholder="Search changes..."
          leftIcon={<InputSearch size="sm-regular" className="text-grayA-9" />}
          rightIcon={
            filters.searchQuery ? (
              <button
                type="button"
                onClick={() => setFilters((p) => ({ ...p, searchQuery: "" }))}
                className="cursor-pointer"
              >
                <CircleXMark size="sm-regular" className="text-grayA-9 hover:text-grayA-12" />
              </button>
            ) : null
          }
          wrapperClassName="flex-1"
          className="text-xs h-9 rounded-md"
        />
        <Select
          value={filters.level?.toString() || "all"}
          onValueChange={(value) =>
            setFilters((p) => ({
              ...p,
              level: value === "all" ? null : Number(value),
            }))
          }
        >
          <SelectTrigger wrapperClassName="w-[150px] rounded-md" className="h-9 rounded-md">
            <SelectValue placeholder="All levels" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All levels</SelectItem>
            <SelectItem value="3">Breaking only</SelectItem>
            <SelectItem value="2">Warnings only</SelectItem>
            <SelectItem value="1">Info only</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={filters.operation}
          onValueChange={(value) => setFilters((p) => ({ ...p, operation: value }))}
        >
          <SelectTrigger wrapperClassName="w-[150px] rounded-md" className="h-9 rounded-md">
            <SelectValue placeholder="All methods" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All methods</SelectItem>
            {stats.operations.map((op) => (
              <SelectItem key={op} value={op}>
                {op}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {(filters.level !== null || filters.operation !== "all" || filters.searchQuery) && (
          <Button
            type="button"
            className="h-9 px-3"
            onClick={() => setFilters({ level: null, operation: "all", searchQuery: "" })}
          >
            Clear
          </Button>
        )}
      </div>

      {/* Changes list */}
      <div className="px-3 pb-3">
        {filteredChanges.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-xs text-grayA-9">No changes match filters</p>
          </div>
        ) : (
          <div className="flex flex-col gap-1">
            {Object.entries(groupedChanges).map(([path, operations]) => {
              const isExpanded = expandedPaths.has(path);
              return (
                <div
                  key={path}
                  className="border border-gray-4 rounded-md overflow-hidden bg-white dark:bg-black"
                >
                  <button
                    type="button"
                    onClick={() => togglePathExpansion(path)}
                    className="w-full flex items-center justify-between py-2 px-3 text-left bg-grayA-1 hover:bg-grayA-2 transition-colors"
                  >
                    <div className="flex items-center gap-2.5 min-w-0 flex-1">
                      <ChevronDown
                        size="sm-regular"
                        className={cn(
                          "text-grayA-9 shrink-0 transition-transform duration-200",
                          !isExpanded && "-rotate-90",
                        )}
                      />
                      <code className="font-mono text-xs text-grayA-12 truncate">{path}</code>
                    </div>
                    <div className="flex items-center gap-2.5 shrink-0">
                      <span className="text-xs text-grayA-9 font-medium tabular-nums">
                        {Object.values(operations).flat().length} changes
                      </span>
                      <div className="flex items-center gap-1.5">
                        {Object.keys(operations).map((op) => (
                          <Badge key={op} variant="secondary" size="sm" className="text-[10px]">
                            {op}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  </button>
                  {isExpanded && (
                    <div className="border-t border-gray-4 bg-grayA-1 animate-in slide-in-from-top-2 duration-200">
                      {Object.entries(operations).map(([operation, changes]) => (
                        <div
                          key={operation}
                          className="py-2 px-3 border-b border-gray-4 last:border-b-0"
                        >
                          <div className="flex items-center gap-2.5 mb-2">
                            <Badge variant="secondary" size="sm" className="text-[10px]">
                              {operation}
                            </Badge>
                          </div>
                          <div className="space-y-1">
                            {changes.map((change, index) => (
                              <div
                                key={`${change.id}-${index}`}
                                className={`px-2 py-1.5 rounded ${getSeverityColor(change.level)}`}
                              >
                                <div className="flex items-start gap-2.5">
                                  <div className="shrink-0 mt-0.5">
                                    {getSeverityIcon(change.level)}
                                  </div>
                                  <div className="flex-1 min-w-0">
                                    <p className="text-xs text-grayA-12">{change.text}</p>
                                    <div className="mt-1 flex items-center gap-2 flex-wrap">
                                      <code className="text-[10px] text-grayA-10 font-mono">
                                        {change.id}
                                      </code>
                                      {change.operationId && (
                                        <span className="text-[10px] text-grayA-10">
                                          • {change.operationId}
                                        </span>
                                      )}
                                    </div>
                                  </div>
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </>
  );
};
