import { CaretRightOutline, CircleInfoSparkle } from "@unkey/icons";
import type React from "react";
import { InfoTooltip } from "../../info-tooltip";

type SearchExampleTooltipProps = {
  onSelectExample: (query: string) => void;
  exampleQueries?: string[];
};

export const SearchExampleTooltip: React.FC<SearchExampleTooltipProps> = ({
  onSelectExample,
  exampleQueries,
}) => {
  const examples = exampleQueries ?? [
    "Show failed requests today",
    "auth errors in the last 3h",
    "API calls from a path that includes /api/v1/oz",
  ];

  return (
    <InfoTooltip
      content={
        <div>
          <div className="font-medium mb-2 flex items-center gap-2 text-[13px]">
            <span>Try queries like:</span>
            <span className="text-[11px] text-gray-11">(click to use)</span>
          </div>
          <ul className="space-y-1.5 pl-1 [&_svg]:size-[10px] ">
            {examples.map((example) => (
              <li key={example} className="flex items-center gap-2">
                <CaretRightOutline className="text-accent-9" />
                <button
                  type="button"
                  className="hover:text-accent-11 transition-colors cursor-pointer hover:underline"
                  onClick={() => onSelectExample(example)}
                  data-testid={`example-${example}`}
                >
                  "{example}"
                </button>
              </li>
            ))}
          </ul>
        </div>
      }
      delayDuration={150}
    >
      <div data-testid="info-icon">
        <CircleInfoSparkle className="size-4 text-accent-9" />
      </div>
    </InfoTooltip>
  );
};
