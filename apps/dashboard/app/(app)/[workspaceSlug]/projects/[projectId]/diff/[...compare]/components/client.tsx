"use client";

import type { ChangelogEntry } from "@unkey/proto";
import {
  AlertTriangle,
  BarChart3,
  ChevronDown,
  ChevronRight,
  Clock,
  Copy,
  Edit,
  ExternalLink,
  FileText,
  Filter,
  Info,
  Layers,
  Minus,
  Plus,
  Search,
  X,
} from "lucide-react";
import type React from "react";
import { useMemo, useState } from "react";
interface DiffViewerProps {
  changelog: ChangelogEntry[];
  fromDeployment?: string;
  toDeployment?: string;
}

type ViewMode = "changes" | "side-by-side" | "timeline";

export const DiffViewer: React.FC<DiffViewerProps> = ({
  changelog,
  fromDeployment = "v1",
  toDeployment = "v2",
}) => {
  // Early return if no diffData
  if (!changelog) {
    return (
      <div className="p-8 text-center">
        <AlertTriangle className="w-12 h-12 text-warn mx-auto mb-4" />
        <h3 className="text-lg font-semibold text-content mb-2">No Diff Data Available</h3>
        <p className="text-content-subtle">Unable to load the comparison data.</p>
      </div>
    );
  }

  const [viewMode, setViewMode] = useState<ViewMode>("changes");
  const [selectedChange, setSelectedChange] = useState<string | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [selectedOperation, setSelectedOperation] = useState<string | null>(null);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(
    new Set(["/users", "/users/{userId}"]),
  );
  const [filters, setFilters] = useState({
    level: null as number | null,
    operation: "all",
    searchQuery: "",
  });
  const [showFilters, setShowFilters] = useState(false);

  // Statistics
  const stats = useMemo(() => {
    const breaking = changelog.filter((c) => c.level === 3).length;
    const warning = changelog.filter((c) => c.level === 2).length;
    const info = changelog.filter((c) => c.level === 1).length;
    const total = changelog.length;

    const operations = [...new Set(changelog.map((c) => c.operation))];
    const paths = [...new Set(changelog.map((c) => c.path))];

    return { breaking, warning, info, total, operations, paths };
  }, [changelog]);

  // Filter changes
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

  // Group changes by path and operation
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

  // Helper functions
  const togglePathExpansion = (path: string) => {
    const newExpanded = new Set(expandedPaths);
    if (newExpanded.has(path)) {
      newExpanded.delete(path);
    } else {
      newExpanded.add(path);
    }
    setExpandedPaths(newExpanded);
  };

  const getSeverityIcon = (level: number) => {
    switch (level) {
      case 3:
        return <AlertTriangle className="w-4 h-4 text-alert" />;
      case 2:
        return <Info className="w-4 h-4 text-warn" />;
      default:
        return <Info className="w-4 h-4 text-brand" />;
    }
  };

  const getSeverityColor = (level: number) => {
    switch (level) {
      case 3:
        return "border-l-4 border-l-alert bg-red-2 hover:bg-red-3";
      case 2:
        return "border-l-4 border-l-warn bg-amber-2 hover:bg-amber-3";
      default:
        return "border-l-4 border-l-brand bg-background-subtle hover:bg-gray-100";
    }
  };

  const getOperationColor = (operation: string) => {
    const colors: Record<string, string> = {
      GET: "bg-gray-100 text-gray-800 border border-gray-200",
      POST: "bg-brand text-brand-foreground border border-gray-200",
      PUT: "bg-amber-2 text-amber-11 border border-gray-200",
      PATCH: "bg-gray-200 text-gray-700 border border-gray-300",
      DELETE: "bg-red-2 text-red-11 border border-gray-200",
    };
    return colors[operation] || "bg-gray-100 text-gray-800 border border-gray-200";
  };

  const getChangeIcon = (changeId: string) => {
    if (changeId.includes("added") || changeId.includes("new")) {
      return <Plus className="w-4 h-4 text-gray-600" />;
    }
    if (changeId.includes("removed") || changeId.includes("deleted")) {
      return <Minus className="w-4 h-4 text-alert" />;
    }
    if (changeId.includes("changed") || changeId.includes("modified")) {
      return <Edit className="w-4 h-4 text-brand" />;
    }
    return <Info className="w-4 h-4 text-gray-600" />;
  };

  // Code diff helpers
  const getBeforeSpec = (path: string, operation: string) => {
    if (path === "/users" && operation === "GET") {
      return `{
  "get": {
    "summary": "Get all users",
    "parameters": [
      {
        "name": "limit",
        "in": "query",
        "schema": {
          "type": "integer",
          "default": 20
        }
      },
      {
        "name": "offset",
        "in": "query",
        "schema": {
          "type": "integer",
          "default": 0
        }
      }
    ],
    "responses": {
      "200": {
        "content": {
          "application/json": {
            "schema": {
              "properties": {
                "users": { "type": "array" },
                "total": { "type": "integer" },
                "limit": { "type": "integer" },
                "offset": { "type": "integer" }
              }
            }
          }
        }
      }
    }
  }
}`;
    }
    return '{\n  "endpoint": "not found"\n}';
  };

  const getAfterSpec = (path: string, operation: string) => {
    if (path === "/users" && operation === "GET") {
      return `{
  "get": {
    "summary": "Get all users",
    "parameters": [
      {
        "name": "pageSize",
        "in": "query",
        "schema": {
          "type": "integer",
          "default": 10
        }
      },
      {
        "name": "page",
        "in": "query",
        "schema": {
          "type": "integer",
          "default": 1
        }
      },
      {
        "name": "status",
        "in": "query",
        "required": true,
        "schema": {
          "type": "string",
          "enum": ["active", "inactive", "suspended"]
        }
      }
    ],
    "responses": {
      "200": {
        "content": {
          "application/json": {
            "schema": {
              "properties": {
                "data": { "type": "array" },
                "pagination": { "type": "object" }
              }
            }
          }
        }
      }
    }
  }
}`;
    }
    return '{\n  "endpoint": "not found"\n}';
  };

  const renderCodeLine = (
    lineContent: string,
    lineNumber: number,
    type: "none" | "added" | "removed" | "modified" = "none",
  ) => {
    const getLineClasses = () => {
      switch (type) {
        case "added":
          return "bg-gray-50 border-l-4 border-l-gray-400";
        case "removed":
          return "bg-red-2 border-l-4 border-l-alert";
        case "modified":
          return "bg-amber-2 border-l-4 border-l-warn";
        default:
          return "bg-background";
      }
    };

    const getLineIcon = () => {
      switch (type) {
        case "added":
          return <Plus className="w-3 h-3 text-gray-600" />;
        case "removed":
          return <Minus className="w-3 h-3 text-alert" />;
        case "modified":
          return <Edit className="w-3 h-3 text-warn" />;
        default:
          return null;
      }
    };

    return (
      <div className={`flex items-start font-mono text-sm ${getLineClasses()}`}>
        <div className="w-8 text-content-subtle text-right pr-2 py-1 select-none">{lineNumber}</div>
        <div className="flex items-center w-6 justify-center py-1">{getLineIcon()}</div>
        <div className="flex-1 py-1 pr-4">
          <code className="whitespace-pre text-content">{lineContent}</code>
        </div>
      </div>
    );
  };

  const renderCodeDiff = (
    beforeCode: string,
    afterCode: string,
    path: string,
    operation: string,
  ) => {
    const beforeLines = beforeCode.split("\n");
    const afterLines = afterCode.split("\n");

    const getDiffType = (_lineIndex: number, side: "before" | "after", line: string) => {
      if (path === "/users" && operation === "GET") {
        if (side === "before") {
          if (line.includes('"limit"') || line.includes('"offset"')) {
            return "removed";
          }
          if (line.includes('"users"') || line.includes('"total"')) {
            return "removed";
          }
        } else {
          if (line.includes('"pageSize"') || line.includes('"page"') || line.includes('"status"')) {
            return "added";
          }
          if (line.includes('"data"') || line.includes('"pagination"')) {
            return "added";
          }
        }
      }
      return "none";
    };

    return (
      <div className="grid grid-cols-2 gap-0 border border-gray-200 rounded-lg overflow-hidden">
        <div className="border-r border-gray-200">
          <div className="bg-gray-50 px-4 py-3 text-sm font-medium text-content border-b border-gray-200">
            Before ({fromDeployment})
          </div>
          <div className="bg-background max-h-96 overflow-y-auto">
            {beforeLines.map((line, index) =>
              renderCodeLine(line, index + 1, getDiffType(index, "before", line)),
            )}
          </div>
        </div>
        <div>
          <div className="bg-gray-50 px-4 py-3 text-sm font-medium text-content border-b border-gray-200">
            After ({toDeployment})
          </div>
          <div className="bg-background max-h-96 overflow-y-auto">
            {afterLines.map((line, index) =>
              renderCodeLine(line, index + 1, getDiffType(index, "after", line)),
            )}
          </div>
        </div>
      </div>
    );
  };

  // View components
  const renderSideBySideView = () => (
    <div className="bg-white rounded-lg border border-gray-200 h-[calc(100vh-300px)] flex flex-col">
      <div className="flex items-center justify-between p-4 border-b border-gray-100 flex-shrink-0">
        <div className="flex items-center space-x-4">
          <h3 className="text-lg font-semibold text-content">Code Diff Comparison</h3>
          {selectedPath && selectedOperation && (
            <div className="flex items-center space-x-2 text-sm text-content-subtle">
              <span
                className={`px-2 py-1 rounded text-xs font-medium ${getOperationColor(
                  selectedOperation,
                )}`}
              >
                {selectedOperation}
              </span>
              <code className="bg-background-subtle px-2 py-1 rounded font-mono">
                {selectedPath}
              </code>
            </div>
          )}
        </div>
        <div className="flex items-center space-x-2">
          <button
            type="button"
            className="p-2 text-content-subtle hover:text-content transition-colors"
            title="Copy diff to clipboard"
            onClick={() => {
              if (typeof window !== "undefined" && selectedPath && selectedOperation) {
                const beforeCode = getBeforeSpec(selectedPath, selectedOperation);
                const afterCode = getAfterSpec(selectedPath, selectedOperation);
                if (navigator.clipboard) {
                  navigator.clipboard.writeText(`BEFORE:\n${beforeCode}\n\nAFTER:\n${afterCode}`);
                }
              }
            }}
          >
            <Copy className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {selectedPath && selectedOperation ? (
          <div className="p-6">
            <div className="mb-4 p-3 bg-background-subtle rounded-lg border border-border">
              <div className="flex items-center space-x-2">
                <FileText className="w-4 h-4 text-content-subtle" />
                <span className="font-mono text-sm text-content">{selectedPath}</span>
                <span
                  className={`px-2 py-1 rounded text-xs font-medium ${getOperationColor(
                    selectedOperation,
                  )}`}
                >
                  {selectedOperation}
                </span>
              </div>
            </div>

            {renderCodeDiff(
              getBeforeSpec(selectedPath, selectedOperation),
              getAfterSpec(selectedPath, selectedOperation),
              selectedPath,
              selectedOperation,
            )}

            <div className="mt-4 p-3 bg-background-subtle rounded-lg">
              <h4 className="text-sm font-medium text-content mb-2">Legend:</h4>
              <div className="flex flex-wrap gap-4 text-xs">
                <div className="flex items-center space-x-1">
                  <Plus className="w-3 h-3 text-gray-600" />
                  <span className="bg-gray-50 px-2 py-1 rounded border-l-4 border-l-gray-400">
                    Added lines
                  </span>
                </div>
                <div className="flex items-center space-x-1">
                  <Minus className="w-3 h-3 text-alert" />
                  <span className="bg-red-2 px-2 py-1 rounded border-l-4 border-l-alert">
                    Removed lines
                  </span>
                </div>
                <div className="flex items-center space-x-1">
                  <Edit className="w-3 h-3 text-warn" />
                  <span className="bg-amber-2 px-2 py-1 rounded border-l-4 border-l-warn">
                    Modified lines
                  </span>
                </div>
              </div>
            </div>

            <div className="mt-4 p-3 bg-background-subtle rounded-lg border-l-4 border-l-brand">
              <h4 className="text-sm font-medium text-content mb-2">Changes for this endpoint:</h4>
              <div className="space-y-1 text-sm">
                {changelog
                  .filter((c) => c.path === selectedPath && c.operation === selectedOperation)
                  .map((change, _index) => (
                    <div
                      key={`${change.id}-${change.path}-${change.operation}`}
                      className="flex items-start space-x-2"
                    >
                      {getSeverityIcon(change.level)}
                      <span className="text-content">{change.text}</span>
                    </div>
                  ))}
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center py-12">
            <Layers className="w-12 h-12 text-content-subtle mx-auto mb-4" />
            <h3 className="text-lg font-medium text-content mb-2">Code Diff Comparison</h3>
            <p className="text-content-subtle mb-4">
              Select a specific endpoint from the Changes view to see the actual code differences.
            </p>
            <div className="flex flex-col sm:flex-row gap-2 justify-center">
              <button
                type="button"
                onClick={() => {
                  setSelectedPath("/users");
                  setSelectedOperation("GET");
                }}
                className="px-4 py-2 bg-brand text-brand-foreground rounded-md hover:bg-brand/90 transition-colors"
              >
                View GET /users Diff
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );

  const renderTimelineView = () => (
    <div className="bg-white rounded-lg border border-gray-200 h-[calc(100vh-300px)] flex flex-col">
      <div className="p-6 border-b border-gray-100 flex-shrink-0">
        <div className="flex items-center space-x-3">
          <Clock className="w-5 h-5 text-brand" />
          <h2 className="text-lg font-semibold text-content">Change Timeline</h2>
          <span className="text-sm text-content-subtle">({filteredChanges.length} changes)</span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="p-6 space-y-6">
          {filteredChanges.map((change, index) => (
            <div key={`${change.id}-${index}`} className="relative">
              {index < filteredChanges.length - 1 && (
                <div className="absolute left-4 top-8 w-0.5 h-16 bg-border" />
              )}
              <div className="flex items-start space-x-4">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${
                    change.level === 3
                      ? "bg-red-2"
                      : change.level === 2
                        ? "bg-amber-2"
                        : "bg-background-subtle"
                  }`}
                >
                  {getSeverityIcon(change.level)}
                </div>
                <div className="flex-1 bg-background-subtle rounded-lg p-4 hover:bg-gray-100 transition-colors cursor-pointer">
                  <div className="flex items-center space-x-2 mb-2">
                    <span
                      className={`px-2 py-1 rounded text-xs font-medium border ${getOperationColor(
                        change.operation,
                      )}`}
                    >
                      {change.operation}
                    </span>
                    <code className="bg-gray-200 px-2 py-1 rounded text-sm font-mono">
                      {change.path}
                    </code>
                    <span
                      className={`px-2 py-1 rounded text-xs ${
                        change.level === 3
                          ? "bg-red-2 text-red-11"
                          : change.level === 2
                            ? "bg-amber-2 text-amber-11"
                            : "bg-background-subtle text-content"
                      }`}
                    >
                      {change.level === 3 ? "Breaking" : change.level === 2 ? "Warning" : "Info"}
                    </span>
                    {getChangeIcon(change.id)}
                  </div>
                  <p className="text-content text-sm mb-2">{change.text}</p>
                  {change.text && (
                    <p className="text-xs text-content-subtle italic bg-background p-2 rounded">
                      {change.text}
                    </p>
                  )}
                  <div className="flex items-center justify-between mt-3">
                    <div className="text-xs text-content-subtle">
                      ID: {change.id} • Section: {change.id}
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedPath(change.path);
                        setSelectedOperation(change.operation);
                        setViewMode("side-by-side");
                      }}
                      className="text-xs text-brand hover:text-brand/80 flex items-center space-x-1 cursor-pointer bg-transparent border-none p-0"
                    >
                      <ExternalLink className="w-3 h-3" />
                      <span>View Details</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}

          {filteredChanges.length === 0 && (
            <div className="text-center py-12">
              <Clock className="w-12 h-12 text-content-subtle mx-auto mb-4" />
              <p className="text-content-subtle">No changes match the current filters</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );

  return (
    <div className="bg-background">
      {/* View Mode Tabs */}
      <div className="bg-white border-b border-gray-100 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="py-3">
            <nav className="flex space-x-6">
              <button
                type="button"
                onClick={() => setViewMode("changes")}
                className={`py-3 px-4 font-medium text-sm transition-colors border-b-2 ${
                  viewMode === "changes"
                    ? "border-brand text-brand"
                    : "border-transparent text-content-subtle hover:text-content hover:border-gray-200"
                }`}
              >
                <div className="flex items-center space-x-2">
                  <FileText className="w-4 h-4" />
                  <span>Changes ({stats.total})</span>
                </div>
              </button>
              <button
                type="button"
                onClick={() => setViewMode("side-by-side")}
                className={`py-3 px-4 font-medium text-sm transition-colors border-b-2 ${
                  viewMode === "side-by-side"
                    ? "border-brand text-brand"
                    : "border-transparent text-content-subtle hover:text-content hover:border-gray-200"
                }`}
              >
                <div className="flex items-center space-x-2">
                  <Layers className="w-4 h-4" />
                  <span>Side-by-Side</span>
                </div>
              </button>
              <button
                type="button"
                onClick={() => setViewMode("timeline")}
                className={`py-3 px-4 font-medium text-sm transition-colors border-b-2 ${
                  viewMode === "timeline"
                    ? "border-brand text-brand"
                    : "border-transparent text-content-subtle hover:text-content hover:border-gray-200"
                }`}
              >
                <div className="flex items-center space-x-2">
                  <Clock className="w-4 h-4" />
                  <span>Timeline</span>
                </div>
              </button>
            </nav>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {viewMode === "changes" && (
          <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
            {/* Summary Panel */}
            <div className="lg:col-span-1">
              <div className="bg-white rounded-lg border border-gray-200 p-6 sticky top">
                <h2 className="text-lg font-semibold text-content mb-4 flex items-center">
                  <BarChart3 className="w-5 h-5 mr-2" />
                  Change Summary
                </h2>

                <div className="space-y-3 mb-6">
                  <div className="flex items-center justify-between p-2 bg-background-subtle rounded">
                    <span className="text-sm text-content-subtle">Total Changes</span>
                    <span className="font-semibold text-lg text-content">{stats.total}</span>
                  </div>
                  <div className="flex items-center justify-between p-2 bg-red-2 rounded">
                    <span className="text-sm text-alert flex items-center">
                      <AlertTriangle className="w-4 h-4 mr-1" />
                      Breaking
                    </span>
                    <span className="font-semibold text-alert text-lg">{stats.breaking}</span>
                  </div>
                  <div className="flex items-center justify-between p-2 bg-amber-2 rounded">
                    <span className="text-sm text-warn flex items-center">
                      <Info className="w-4 h-4 mr-1" />
                      Warning
                    </span>
                    <span className="font-semibold text-warn text-lg">{stats.warning}</span>
                  </div>
                  <div className="flex items-center justify-between p-2 bg-background-subtle rounded">
                    <span className="text-sm text-brand">Endpoints Affected</span>
                    <span className="font-semibold text-brand text-lg">{stats.paths.length}</span>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => setShowFilters(!showFilters)}
                  className="w-full flex items-center justify-between p-3 text-sm font-medium text-content bg-background-subtle rounded-lg hover:bg-gray-100 mb-4 transition-colors"
                >
                  <div className="flex items-center space-x-2">
                    <Filter className="w-4 h-4" />
                    <span>Filters</span>
                  </div>
                  {showFilters ? (
                    <ChevronDown className="w-4 h-4" />
                  ) : (
                    <ChevronRight className="w-4 h-4" />
                  )}
                </button>

                {showFilters && (
                  <div className="space-y-4">
                    <div>
                      <label
                        htmlFor="search-changes"
                        className="block text-sm font-medium text-content mb-2"
                      >
                        Search Changes
                      </label>
                      <div className="relative">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-content-subtle" />
                        <input
                          id="search-changes"
                          type="text"
                          value={filters.searchQuery}
                          onChange={(e) =>
                            setFilters((prev) => ({
                              ...prev,
                              searchQuery: e.target.value,
                            }))
                          }
                          placeholder="Search changes..."
                          className="w-full pl-10 pr-4 py-2 border border-border rounded-md text-sm bg-background text-content focus:ring-2 focus:ring-brand focus:border-brand"
                        />
                        {filters.searchQuery && (
                          <button
                            type="button"
                            onClick={() =>
                              setFilters((prev) => ({
                                ...prev,
                                searchQuery: "",
                              }))
                            }
                            className="absolute right-3 top-1/2 transform -translate-y-1/2 hover:bg-background-subtle rounded p-1"
                          >
                            <X className="w-4 h-4 text-content-subtle hover:text-content" />
                          </button>
                        )}
                      </div>
                    </div>

                    <div>
                      <label
                        htmlFor="severity-level"
                        className="block text-sm font-medium text-content mb-2"
                      >
                        Severity Level
                      </label>
                      <select
                        id="severity-level"
                        value={filters.level || ""}
                        onChange={(e) =>
                          setFilters((prev) => ({
                            ...prev,
                            level: e.target.value ? Number(e.target.value) : null,
                          }))
                        }
                        className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background text-content focus:ring-2 focus:ring-brand focus:border-brand"
                      >
                        <option value="">All Levels</option>
                        <option value="3">Breaking Changes Only</option>
                        <option value="2">Warnings Only</option>
                        <option value="1">Info Only</option>
                      </select>
                    </div>

                    <div>
                      <label
                        htmlFor="http-method"
                        className="block text-sm font-medium text-content mb-2"
                      >
                        HTTP Method
                      </label>
                      <select
                        id="http-method"
                        value={filters.operation}
                        onChange={(e) =>
                          setFilters((prev) => ({
                            ...prev,
                            operation: e.target.value,
                          }))
                        }
                        className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background text-content focus:ring-2 focus:ring-brand focus:border-brand"
                      >
                        <option value="all">All Methods</option>
                        {stats.operations.map((op) => (
                          <option key={op} value={op}>
                            {op}
                          </option>
                        ))}
                      </select>
                    </div>

                    {(filters.level !== null ||
                      filters.operation !== "all" ||
                      filters.searchQuery) && (
                      <button
                        type="button"
                        onClick={() =>
                          setFilters({
                            level: null,
                            operation: "all",
                            searchQuery: "",
                          })
                        }
                        className="w-full px-3 py-2 text-sm text-content-subtle border border-border rounded-md hover:bg-background-subtle transition-colors"
                      >
                        Clear All Filters
                      </button>
                    )}
                  </div>
                )}
              </div>
            </div>

            {/* Changes List */}
            <div className="lg:col-span-3">
              <div className="bg-white rounded-lg border border-gray-200 h-[calc(100vh-300px)] flex flex-col">
                <div className="flex-1 overflow-y-auto p-6">
                  {filteredChanges.length === 0 ? (
                    <div className="text-center py-12">
                      <FileText className="w-12 h-12 text-content-subtle mx-auto mb-4" />
                      <p className="text-content-subtle text-lg">
                        No changes match the current filters
                      </p>
                      <button
                        type="button"
                        onClick={() =>
                          setFilters({
                            level: null,
                            operation: "all",
                            searchQuery: "",
                          })
                        }
                        className="mt-4 px-4 py-2 bg-brand text-brand-foreground rounded-md hover:bg-brand/90 transition-colors"
                      >
                        Clear Filters
                      </button>
                    </div>
                  ) : (
                    <div className="space-y-4">
                      {Object.entries(groupedChanges).map(([path, operations]) => (
                        <div
                          key={path}
                          className="border border-gray-200 rounded-lg overflow-hidden"
                        >
                          <button
                            type="button"
                            onClick={() => togglePathExpansion(path)}
                            className="w-full flex items-center justify-between p-4 text-left hover:bg-background-subtle transition-colors"
                          >
                            <div className="flex items-center space-x-3">
                              {expandedPaths.has(path) ? (
                                <ChevronDown className="w-4 h-4 text-content-subtle" />
                              ) : (
                                <ChevronRight className="w-4 h-4 text-content-subtle" />
                              )}
                              <code className="font-mono text-sm bg-background-subtle px-3 py-1 rounded-md text-content">
                                {path}
                              </code>
                              <span className="text-sm text-content-subtle bg-background-subtle px-2 py-1 rounded-full">
                                {Object.values(operations).flat().length} changes
                              </span>
                            </div>
                            <div className="flex items-center space-x-2">
                              {Object.keys(operations).map((op) => (
                                <span
                                  key={op}
                                  className={`px-2 py-1 rounded text-xs font-medium ${getOperationColor(
                                    op,
                                  )}`}
                                >
                                  {op}
                                </span>
                              ))}
                              <button
                                type="button"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setSelectedPath(path);
                                  setSelectedOperation(Object.keys(operations)[0]);
                                  setViewMode("side-by-side");
                                }}
                                className="p-1 text-content-subtle hover:text-brand transition-colors cursor-pointer"
                                title="View side-by-side comparison"
                              >
                                <Layers className="w-4 h-4" />
                              </button>
                            </div>
                          </button>

                          {expandedPaths.has(path) && (
                            <div className="border-t border-border bg-background-subtle">
                              {Object.entries(operations).map(([operation, changes]) => (
                                <div
                                  key={operation}
                                  className="p-4 border-b border-gray-100 last:border-b-0"
                                >
                                  <div className="flex items-center space-x-2 mb-3">
                                    <span
                                      className={`px-3 py-1 rounded text-xs font-medium ${getOperationColor(
                                        operation,
                                      )}`}
                                    >
                                      {operation}
                                    </span>
                                    <span className="text-sm text-content-subtle bg-background px-2 py-1 rounded">
                                      {changes.length} changes
                                    </span>
                                  </div>

                                  <div className="space-y-3">
                                    {changes.map((change, index) => (
                                      <button
                                        type="button"
                                        key={`${change.path}-${change.operation}-${change.id}-${index}`}
                                        className={`p-3 rounded-r-lg cursor-pointer transition-all duration-200 text-left w-full ${getSeverityColor(
                                          change.level,
                                        )} ${
                                          selectedChange === `${change.id}-${index}`
                                            ? "ring-2 ring-brand shadow-md"
                                            : ""
                                        }`}
                                        onClick={() =>
                                          setSelectedChange(
                                            selectedChange === `${change.id}-${index}`
                                              ? null
                                              : `${change.id}-${index}`,
                                          )
                                        }
                                      >
                                        <div className="flex items-start space-x-3">
                                          <div className="flex items-center space-x-2 flex-shrink-0">
                                            {getSeverityIcon(change.level)}
                                            {getChangeIcon(change.id)}
                                          </div>
                                          <div className="flex-1 min-w-0">
                                            <p className="text-sm text-content font-medium mb-1">
                                              {change.text}
                                            </p>
                                            {change.text && (
                                              <p className="text-xs text-content-subtle italic bg-background/70 p-2 rounded border-l-2 border-gray-300">
                                                💡 {change.text}
                                              </p>
                                            )}
                                            <div className="flex items-center justify-between mt-3">
                                              <div className="flex items-center space-x-4 text-xs text-content-subtle">
                                                <span className="font-mono bg-gray-200 px-2 py-1 rounded">
                                                  ID: {change.id}
                                                </span>
                                                <span>Section: {change.id}</span>
                                              </div>
                                              <button
                                                type="button"
                                                onClick={(e) => {
                                                  e.stopPropagation();
                                                  setSelectedPath(change.path);
                                                  setSelectedOperation(change.operation);
                                                  setViewMode("side-by-side");
                                                }}
                                                className="text-xs text-brand hover:text-brand/80 flex items-center space-x-1 bg-background px-2 py-1 rounded border hover:shadow-sm transition-all cursor-pointer"
                                              >
                                                <ExternalLink className="w-3 h-3" />
                                                <span>Compare</span>
                                              </button>
                                            </div>
                                          </div>
                                        </div>
                                      </button>
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
              </div>
            </div>
          </div>
        )}

        {viewMode === "side-by-side" && renderSideBySideView()}
        {viewMode === "timeline" && renderTimelineView()}
      </div>
    </div>
  );
};
