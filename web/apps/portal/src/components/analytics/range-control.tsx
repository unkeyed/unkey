import type { AnalyticsPeriod } from "./schema/analytics.schema";

type Props = {
  options: AnalyticsPeriod[];
  value: number;
  onChange: (days: number) => void;
};

function shortLabel(days: number): string {
  return days === 1 ? "24h" : `${days}d`;
}

export function RangeControl({ options, value, onChange }: Props) {
  return (
    <fieldset
      className="inline-flex items-center gap-0.5 rounded-md bg-gray-3 p-0.5"
      aria-label="Time range"
    >
      {options.map((option) => {
        const active = option.days === value;
        return (
          <button
            key={option.days}
            type="button"
            aria-pressed={active}
            onClick={() => onChange(option.days)}
            className={`min-h-8 rounded-sm px-3 py-1.5 font-medium text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-offset-1 sm:min-h-0 sm:py-1 ${
              active ? "bg-background text-gray-12 shadow-xs" : "text-gray-11 hover:text-gray-12"
            }`}
          >
            {shortLabel(option.days)}
          </button>
        );
      })}
    </fieldset>
  );
}
