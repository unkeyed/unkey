import { ChevronExpandY } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";

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
        className="bg-transparent rounded-full flex items-center gap-1.5 border-0 h-auto !min-h-0 !p-0 focus:border-none focus:ring-0 hover:bg-grayA-2 transition-colors justify-normal "
        rightIcon={<ChevronExpandY className="text-accent-8 size-3.5" />}
      >
        <span className="text-gray-11 text-xs">{label}</span>
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="min-w-[80px]">
        {options.map((option) => (
          <SelectItem
            key={option}
            value={option}
            className="cursor-pointer hover:bg-grayA-3 data-[highlighted]:bg-grayA-2 font-mono font-medium text-sm"
          >
            {option}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
