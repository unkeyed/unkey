import { ChevronExpandY } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger } from "@unkey/ui";

type MetricSelectProps = {
  label: string;
  value: string;
  options: string[];
  onValueChange?: (value: string) => void;
};

export function MetricSelect({ label, value, options, onValueChange }: MetricSelectProps) {
  return (
    <Select defaultValue={value} onValueChange={onValueChange}>
      <SelectTrigger
        wrapperClassName="w-fit shrink-0"
        className="h-auto min-h-0! border-0 bg-transparent shadow-none text-gray-12 text-[13px] gap-1 !p-0 !pr-4 focus:ring-0 focus:outline-none focus:border-0 focus-visible:ring-0 focus-visible:outline-none hover:text-gray-11 transition-colors"
        rightIcon={<ChevronExpandY className="text-grayA-8 size-3" />}
      >
        <span className="text-gray-12">{label}</span>
        <span className="text-grayA-10 text-[11px] font-mono">{value}</span>
      </SelectTrigger>
      <SelectContent className="min-w-[80px]">
        {options.map((option) => (
          <SelectItem
            key={option}
            value={option}
            className="cursor-pointer hover:bg-grayA-3 data-highlighted:bg-grayA-2 font-mono font-medium text-xs"
          >
            {option}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
