import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { useGatewayLogsFilters } from "../../../../../hooks/use-gateway-logs-filters";

type MethodOption = {
  id: number;
  method: string;
  checked: boolean;
};

const options: MethodOption[] = [
  { id: 1, method: "GET", checked: false },
  { id: 2, method: "POST", checked: false },
  { id: 3, method: "PUT", checked: false },
  { id: 4, method: "DELETE", checked: false },
  { id: 5, method: "PATCH", checked: false },
] as const;

export const GatewayMethodsFilter = () => {
  const { filters, updateFilters } = useGatewayLogsFilters();
  return (
    <FilterCheckbox
      options={options}
      filterField="methods"
      checkPath="method"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.method}</div>
      )}
      createFilterValue={(option) => ({
        value: option.method,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
