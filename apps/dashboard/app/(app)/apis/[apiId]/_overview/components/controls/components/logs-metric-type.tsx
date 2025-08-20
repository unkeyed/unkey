"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { ChevronDown } from "@unkey/icons";
import { Button } from "@unkey/ui";
import {
  METRIC_TYPE_LABELS,
  type MetricType,
  useMetricType,
} from "../../../hooks/use-metric-type";

export const LogsMetricType = () => {
  const { metricType, setMetricType } = useMetricType();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="md"
          className={cn("px-2 rounded-lg gap-1", metricType !== "requests" ? "bg-gray-4" : "")}
          aria-label="Select metric type"
        >
          <span className="text-gray-12 font-medium text-[13px]">
            {METRIC_TYPE_LABELS[metricType]}
          </span>
          <ChevronDown className="text-gray-9 size-3" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[140px]">
        {(Object.keys(METRIC_TYPE_LABELS) as MetricType[]).map((type) => (
          <DropdownMenuItem
            key={type}
            onClick={() => setMetricType(type)}
            className={cn("cursor-pointer", metricType === type && "bg-gray-3")}
          >
            {METRIC_TYPE_LABELS[type]}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
