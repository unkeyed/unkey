import { ControlPill } from "@unkey/ui";
import { useState } from "react";

const FILTERS = [
  { id: "region", field: "Region", operator: "is" as const, value: "Frankfurt" },
  { id: "status", field: "Status", operator: "is" as const, value: "Running" },
];

export default function ControlPillVariants() {
  const [filters, setFilters] = useState(FILTERS);
  const remove = (id: string) => setFilters(filters.filter((f) => f.id !== id));

  return (
    <div className="flex flex-col items-start gap-4 text-xs">
      {(["solid", "outline"] as const).map((variant) => (
        <div key={variant} className="flex flex-wrap items-center gap-2">
          {filters.map((filter, index) => (
            <ControlPill
              key={filter.id}
              variant={variant}
              index={index}
              filter={filter}
              formatFieldName={(field) => field}
              formatValue={(value) => String(value)}
              onRemove={remove}
            />
          ))}
        </div>
      ))}
    </div>
  );
}
