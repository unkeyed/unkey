import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useFilters } from "../../../../hooks/use-filters";

type OutcomeOption = {
  id: number;
  outcome: string;
  display: string;
  label: string;
  color: string;
  checked: boolean;
};

const getOutcomeColor = (outcome: string): string => {
  switch (outcome) {
    case "VALID":
      return "bg-success-9";
    case "RATE_LIMITED":
      return "bg-warning-9";
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
      return "bg-error-9";
    case "DISABLED":
      return "bg-gray-9";
    case "EXPIRED":
      return "bg-orange-9";
    case "USAGE_EXCEEDED":
      return "bg-feature-9";
    default:
      return "bg-accent-9";
  }
};

const options: OutcomeOption[] = [
  {
    id: 1,
    outcome: "VALID",
    display: "Valid",
    label: "Valid",
    color: "bg-success-9",
    checked: false,
  },
  {
    id: 2,
    outcome: "RATE_LIMITED",
    display: "Rate Limited",
    label: "Rate Limited",
    color: "bg-warning-9",
    checked: false,
  },
  {
    id: 3,
    outcome: "INSUFFICIENT_PERMISSIONS",
    display: "Insufficient Permissions",
    label: "Insufficient Permissions",
    color: "bg-error-9",
    checked: false,
  },
  {
    id: 4,
    outcome: "FORBIDDEN",
    display: "Forbidden",
    label: "Forbidden",
    color: "bg-error-9",
    checked: false,
  },
  {
    id: 5,
    outcome: "DISABLED",
    display: "Disabled",
    label: "Disabled",
    color: "bg-gray-9",
    checked: false,
  },
  {
    id: 6,
    outcome: "EXPIRED",
    display: "Expired",
    label: "Expired",
    color: "bg-orange-9",
    checked: false,
  },
  {
    id: 7,
    outcome: "USAGE_EXCEEDED",
    display: "Usage Exceeded",
    label: "Usage Exceeded",
    color: "bg-feature-9",
    checked: false,
  },
];

export const OutcomesFilter = () => {
  const { filters, updateFilters } = useFilters();

  return (
    <FilterCheckbox
      options={options}
      filterField="outcomes"
      checkPath="outcome"
      renderOptionContent={(checkbox) => (
        <>
          <div className={`size-2 ${checkbox.color} rounded-[2px]`} />
          <span className="text-accent-12 text-xs">{checkbox.label}</span>
        </>
      )}
      createFilterValue={(option) => ({
        value: option.outcome,
        metadata: {
          colorClass: getOutcomeColor(option.outcome),
        },
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
