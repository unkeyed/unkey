"use client";
import {
  ChevronDown,
  ChevronRight,
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
    return <CircleInfo size="sm-regular" className="text-gray-9" />;
  };

  const getSeverityColor = (level: number) => {
    if (level === 3) {
      return "border-l-2 border-l-error-9 bg-errorA-2";
    }
    if (level === 2) {
      return "border-l-2 border-l-warning-9 bg-warningA-2";
    }
    return "border-l-2 border-l-gray-6 bg-grayA-1";
  };

  if (!changelog || changelog.length === 0) {
    return (
      <div className="text-center py-12 mx-3 mb-3">
        <p className="text-xs text-gray-9">
          No differences between {fromDeployment} and {toDeployment}
        </p>
      </div>
    );
  }

  return (
    <>
      {/* Stats header - integrated into parent card */}
      <div className="px-3 pb-3">
        <div className="flex justify-between items-center">
          <div className="flex flex-col gap-1">
            <div className="text-accent-12 font-medium text-xs">API Changes</div>
            <div className="text-gray-9 text-xs">
              {stats.total} changes • {stats.paths.length} endpoints
            </div>
          </div>
          <div className="flex items-center gap-4">
            {stats.breaking > 0 && (
              <Badge variant="error" className="gap-1.5">
                <TriangleWarning size="sm-regular" className="shrink-0" />
                {stats.breaking} breaking
              </Badge>
            )}
            {stats.warning > 0 && (
              <Badge variant="warning" className="gap-1.5 p-1.5">
                <CircleWarning size="sm-regular" className="shrink-0" />
                {stats.warning} warnings
              </Badge>
            )}
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="px-3 pb-2 flex gap-2 items-center">
        <Input
          type="text"
          value={filters.searchQuery}
          onChange={(e) => setFilters((p) => ({ ...p, searchQuery: e.target.value }))}
          placeholder="Search changes..."
          leftIcon={<InputSearch size="sm-regular" className="text-gray-9" />}
          rightIcon={
            filters.searchQuery ? (
              <button
                type="button"
                onClick={() => setFilters((p) => ({ ...p, searchQuery: "" }))}
                className="cursor-pointer"
              >
                <CircleXMark size="sm-regular" className="text-gray-9 hover:text-gray-12" />
              </button>
            ) : null
          }
          wrapperClassName="flex-1"
          className="text-xs h-9 min-w-[500px] rounded-md"
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
            <p className="text-xs text-gray-9">No changes match filters</p>
          </div>
        ) : (
          <div className="flex flex-col gap-1">
            {Object.entries(groupedChanges).map(([path, operations]) => (
              <div key={path} className="border border-gray-4 rounded-md overflow-hidden bg-white">
                <button
                  type="button"
                  onClick={() => togglePathExpansion(path)}
                  className="w-full flex items-center justify-between py-2 px-3 text-left bg-gray-1 hover:bg-grayA-2 transition-colors"
                >
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    {expandedPaths.has(path) ? (
                      <ChevronDown size="sm-regular" className="text-gray-9 shrink-0" />
                    ) : (
                      <ChevronRight size="sm-regular" className="text-gray-9 shrink-0" />
                    )}
                    <code className="font-mono text-xs text-gray-12 truncate">{path}</code>
                    <span className="text-xs text-gray-9 shrink-0">
                      {Object.values(operations).flat().length}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {Object.keys(operations).map((op) => (
                      <Badge key={op} variant="secondary" size="sm" className="text-[10px]">
                        {op}
                      </Badge>
                    ))}
                  </div>
                </button>
                {expandedPaths.has(path) && (
                  <div className="border-t border-gray-4 bg-gray-1">
                    {Object.entries(operations).map(([operation, changes]) => (
                      <div
                        key={operation}
                        className="py-2 px-3 border-b border-gray-4 last:border-b-0"
                      >
                        <div className="flex items-center gap-2 mb-2">
                          <Badge variant="secondary" size="sm" className="text-[10px]">
                            {operation}
                          </Badge>
                          <span className="text-xs text-gray-9">{changes.length}</span>
                        </div>
                        <div className="space-y-1">
                          {changes.map((change, index) => (
                            <div
                              key={`${change.id}-${index}`}
                              className={`px-2 py-1.5 rounded ${getSeverityColor(change.level)}`}
                            >
                              <div className="flex items-start gap-2">
                                <div className="shrink-0 mt-0.5">
                                  {getSeverityIcon(change.level)}
                                </div>
                                <div className="flex-1 min-w-0">
                                  <p className="text-xs text-gray-12">{change.text}</p>
                                  <div className="mt-1 flex items-center gap-2 flex-wrap">
                                    <code className="text-[10px] text-gray-10 font-mono">
                                      {change.id}
                                    </code>
                                    {change.operationId && (
                                      <span className="text-[10px] text-gray-10">
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
            ))}
          </div>
        )}
      </div>
    </>
  );
};
