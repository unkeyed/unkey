import type { LogsFilterValue } from "@/lib/schemas/logs.filter.schema";
import { Button, Checkbox } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useState } from "react";
import { useSentinelLogsFilters } from "../../../../../hooks/use-sentinel-logs-filters";

type StatusOption = {
  id: number;
  status: number;
  display: string;
  label: string;
  color: string;
  checked: boolean;
};

const RANGE_OPTIONS: StatusOption[] = [
  { id: 1, status: 200, display: "2xx", label: "Success", color: "bg-success-9", checked: false },
  { id: 2, status: 300, display: "3xx", label: "Redirect", color: "bg-info-8", checked: false },
  { id: 3, status: 400, display: "4xx", label: "Warning", color: "bg-warning-8", checked: false },
  { id: 4, status: 500, display: "5xx", label: "Error", color: "bg-error-9", checked: false },
];

const RANGE_VALUES = new Set([200, 300, 400, 500]);

export const SentinelStatusFilter = () => {
  const { filters, updateFilters } = useSentinelLogsFilters();

  const [checkboxes, setCheckboxes] = useState<StatusOption[]>(() => {
    const activeStatuses = new Set(
      filters.filter((f) => f.field === "status").map((f) => Number(f.value)),
    );
    return RANGE_OPTIONS.map((opt) => ({ ...opt, checked: activeStatuses.has(opt.status) }));
  });

  const [customCode, setCustomCode] = useState(() => {
    const exact = filters.find((f) => f.field === "status" && !RANGE_VALUES.has(Number(f.value)));
    return exact ? String(exact.value) : "";
  });
  const [codeError, setCodeError] = useState(false);

  const codeMeta = metaForCode(customCode);

  const handleCheckboxToggle = useCallback((index: number) => {
    setCustomCode("");
    setCodeError(false);
    setCheckboxes((prev) => prev.map((c, i) => (i === index ? { ...c, checked: !c.checked } : c)));
  }, []);

  const handleSelectAll = useCallback(() => {
    setCustomCode("");
    setCodeError(false);
    setCheckboxes((prev) => {
      const allChecked = prev.every((c) => c.checked);
      return prev.map((c) => ({ ...c, checked: !allChecked }));
    });
  }, []);

  const handleCustomCodeChange = useCallback((value: string) => {
    setCustomCode(value);
    setCodeError(false);
    if (value) {
      setCheckboxes((prev) => {
        if (!prev.some((c) => c.checked)) {
          return prev;
        }
        return prev.map((c) => ({ ...c, checked: false }));
      });
    }
  }, []);

  const handleApply = useCallback(() => {
    const otherFilters = filters.filter((f) => f.field !== "status");

    const rangeFilters: LogsFilterValue[] = checkboxes
      .filter((c) => c.checked)
      .map((c) => ({
        id: crypto.randomUUID(),
        field: "status",
        operator: "is",
        value: c.status,
        metadata: { colorClass: c.color },
      }));

    const exactFilters: LogsFilterValue[] = [];
    if (customCode !== "") {
      if (!codeMeta.label) {
        setCodeError(true);
        return;
      }
      exactFilters.push({
        id: crypto.randomUUID(),
        field: "status",
        operator: "is",
        value: Number(customCode),
        metadata: { colorClass: codeMeta.color },
      });
    }

    updateFilters([...otherFilters, ...rangeFilters, ...exactFilters]);
  }, [filters, checkboxes, customCode, updateFilters, codeMeta]);

  const allChecked = checkboxes.every((c) => c.checked);

  return (
    <div className="flex flex-col p-2">
      <div className="flex flex-col gap-2 font-mono px-2 py-2">
        <label htmlFor="checkbox-999" className="flex items-center gap-[18px] cursor-pointer">
          <Checkbox
            id="checkbox-999"
            checked={allChecked}
            className="size-4 rounded-sm border-gray-4 [&_svg]:size-3"
            onClick={(e) => {
              e.stopPropagation();
              handleSelectAll();
            }}
          />
          <span className="text-xs text-accent-12">
            {allChecked ? "Unselect All" : "Select All"}
          </span>
        </label>

        {checkboxes.map((checkbox, index) => (
          <label
            key={checkbox.id}
            htmlFor={`checkbox-${checkbox.id}`}
            className="flex gap-[18px] items-center py-1 cursor-pointer"
          >
            <Checkbox
              id={`checkbox-${checkbox.id}`}
              checked={checkbox.checked}
              className="size-4 rounded-sm border-gray-4 [&_svg]:size-3"
              onClick={(e) => {
                e.stopPropagation();
                handleCheckboxToggle(index);
              }}
            />
            <div className={cn("size-2 rounded-[2px]", checkbox.color)} />
            <span className="text-accent-9 text-xs">{checkbox.display}</span>
            <span className="text-accent-12 text-xs">{checkbox.label}</span>
          </label>
        ))}

        <div className="flex gap-[18px] items-center py-1">
          <div className="size-4 shrink-0" />
          <div
            className={cn(
              "size-2 rounded-[2px] shrink-0",
              codeError ? "bg-error-9" : codeMeta.color,
            )}
          />
          <input
            type="number"
            min={100}
            max={599}
            step={1}
            value={customCode}
            onChange={(e) => handleCustomCodeChange(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handleApply();
              }
            }}
            placeholder="418"
            className="text-accent-9 text-xs bg-transparent border-b border-gray-6 outline-none w-[3ch] font-mono placeholder:text-accent-9/40 focus:border-accent-9 [&::-webkit-inner-spin-button]:appearance-none"
          />
          {codeError ? (
            <span className="text-error-9 text-xs">100–599</span>
          ) : codeMeta.label ? (
            <span className="text-accent-12 text-xs">{codeMeta.label}</span>
          ) : (
            <span className="text-accent-9/40 text-xs">Custom</span>
          )}
        </div>
      </div>

      <Button
        variant="primary"
        className="mt-2 w-full h-9 rounded-md focus:ring-4 focus:ring-accent-9 focus:ring-offset-2"
        onClick={handleApply}
      >
        Apply Filter
      </Button>
    </div>
  );
};

function metaForCode(input: string): { color: string; label: string } {
  const code = Number(input);
  if (!input || !Number.isInteger(code) || code < 100 || code > 599) {
    return { color: "bg-gray-5", label: "" };
  }
  if (code >= 500) {
    return { color: "bg-error-9", label: "Error" };
  }
  if (code >= 400) {
    return { color: "bg-warning-8", label: "Warning" };
  }
  if (code >= 300) {
    return { color: "bg-info-8", label: "Redirect" };
  }
  return { color: "bg-success-9", label: "Success" };
}
