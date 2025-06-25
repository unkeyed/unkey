"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { ControlCloud } from "@unkey/ui";
import type { FilterOperator } from "@unkey/ui/src/validation/filter.types";
import { useEffect, useState } from "react";

// Define FilterValue type locally for examples
type FilterValue = {
  id: string;
  field: string;
  operator: FilterOperator;
  value: string | number;
  metadata?: {
    colorClass?: string;
    icon?: React.ReactNode;
  };
};

// Mock filter data for examples
const createMockFilter = (
  field: string,
  operator: FilterOperator,
  value: string | number,
  id?: string,
): FilterValue => ({
  id: id || crypto.randomUUID(),
  field,
  operator,
  value,
});

export function BasicControlCloud() {
  const [filters, setFilters] = useState<FilterValue[]>([
    createMockFilter("status", "is", "200"),
    createMockFilter("method", "is", "GET"),
  ]);

  const removeFilter = (id: string) => {
    setFilters(filters.filter((f) => f.id !== id));
  };

  const updateFilters = (newFilters: FilterValue[]) => {
    setFilters(newFilters);
  };

  const formatFieldName = (field: string): string => {
    switch (field) {
      case "status":
        return "Status";
      case "method":
        return "Method";
      case "path":
        return "Path";
      default:
        return field.charAt(0).toUpperCase() + field.slice(1);
    }
  };

  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-2xl border border-gray-6 rounded-lg">
        <ControlCloud
          filters={filters}
          removeFilter={removeFilter}
          updateFilters={updateFilters}
          formatFieldName={formatFieldName}
        />
        <div className="p-4 text-sm text-gray-11">Active filters: {filters.length}</div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function TimeBasedFilters() {
  const [filters, setFilters] = useState<FilterValue[]>([
    createMockFilter("startTime", "is", Date.now() - 3600000), // 1 hour ago
    createMockFilter("endTime", "is", Date.now()),
  ]);

  const removeFilter = (id: string) => {
    setFilters(filters.filter((f) => f.id !== id));
  };

  const updateFilters = (newFilters: FilterValue[]) => {
    setFilters(newFilters);
  };

  const formatFieldName = (field: string): string => {
    switch (field) {
      case "startTime":
        return "Start time";
      case "endTime":
        return "End time";
      default:
        return field.charAt(0).toUpperCase() + field.slice(1);
    }
  };

  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-2xl border border-gray-6 rounded-lg">
        <ControlCloud
          filters={filters}
          removeFilter={removeFilter}
          updateFilters={updateFilters}
          formatFieldName={formatFieldName}
        />
        <div className="p-4 text-sm text-gray-11">Time range filters applied</div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function MultipleFilterTypes() {
  const [filters, setFilters] = useState<FilterValue[]>([
    createMockFilter("status", "is", "404"),
    createMockFilter("method", "is", "POST"),
    createMockFilter("path", "contains", "/api/users"),
    createMockFilter("duration", "is", 1000),
  ]);

  const removeFilter = (id: string) => {
    setFilters(filters.filter((f) => f.id !== id));
  };

  const updateFilters = (newFilters: FilterValue[]) => {
    setFilters(newFilters);
  };

  const formatFieldName = (field: string): string => {
    switch (field) {
      case "status":
        return "Status";
      case "method":
        return "Method";
      case "path":
        return "Path";
      case "duration":
        return "Duration";
      default:
        return field.charAt(0).toUpperCase() + field.slice(1);
    }
  };

  const formatValue = (value: string | number, field: string): string => {
    if (field === "duration") {
      return `${value}ms`;
    }
    return String(value);
  };

  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-2xl border border-gray-6 rounded-lg">
        <ControlCloud
          filters={filters}
          removeFilter={removeFilter}
          updateFilters={updateFilters}
          formatFieldName={formatFieldName}
          formatValue={formatValue}
        />
        <div className="p-4 text-sm text-gray-11">
          Mixed filter types: status, method, path, and duration
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function EmptyState() {
  const [filters, setFilters] = useState<FilterValue[]>([]);

  const removeFilter = (id: string) => {
    setFilters(filters.filter((f) => f.id !== id));
  };

  const updateFilters = (newFilters: FilterValue[]) => {
    setFilters(newFilters);
  };

  const formatFieldName = (field: string): string => {
    return field.charAt(0).toUpperCase() + field.slice(1);
  };

  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-2xl border border-gray-6 rounded-lg">
        <ControlCloud
          filters={filters}
          removeFilter={removeFilter}
          updateFilters={updateFilters}
          formatFieldName={formatFieldName}
        />
        <div className="p-4 text-sm text-gray-11">
          No filters applied (component is hidden when empty)
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function InteractiveExample() {
  const [filters, setFilters] = useState<FilterValue[]>([]);
  const [isMac, setIsMac] = useState(false);

  // Client-side platform detection
  useEffect(() => {
    setIsMac(/(Mac|iPhone|iPod|iPad)/i.test(navigator.platform));
  }, []);

  const removeFilter = (id: string) => {
    setFilters(filters.filter((f) => f.id !== id));
  };

  const updateFilters = (newFilters: FilterValue[]) => {
    setFilters(newFilters);
  };

  const addFilter = (field: string, value: string | number) => {
    const newFilter = createMockFilter(field, "is", value);
    setFilters([...filters, newFilter]);
  };

  const formatFieldName = (field: string): string => {
    switch (field) {
      case "status":
        return "Status";
      case "method":
        return "Method";
      case "path":
        return "Path";
      default:
        return field.charAt(0).toUpperCase() + field.slice(1);
    }
  };

  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-2xl space-y-4">
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => addFilter("status", "200")}
            className="px-3 py-1 text-xs bg-gray-3 rounded hover:bg-gray-4"
          >
            Add Status Filter
          </button>
          <button
            type="button"
            onClick={() => addFilter("method", "GET")}
            className="px-3 py-1 text-xs bg-gray-3 rounded hover:bg-gray-4"
          >
            Add Method Filter
          </button>
          <button
            type="button"
            onClick={() => addFilter("path", "/api/users")}
            className="px-3 py-1 text-xs bg-gray-3 rounded hover:bg-gray-4"
          >
            Add Path Filter
          </button>
        </div>
        <div className="border border-gray-6 rounded-lg">
          <ControlCloud
            filters={filters}
            removeFilter={removeFilter}
            updateFilters={updateFilters}
            formatFieldName={formatFieldName}
          />
          <div className="p-4 text-sm text-gray-11">
            Click buttons above to add filters, then use keyboard navigation (
            {isMac ? "⌥+⇧+C" : "Alt+Shift+C"} to focus)
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
