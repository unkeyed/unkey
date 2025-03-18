import { CaretRightOutline, CircleInfoSparkle } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "components/ui/tooltip";

type SearchExampleTooltipProps = {
  onSelectExample: (query: string) => void;
  exampleQueries?: { id: string; text: string }[];
};

export const SearchExampleTooltip: React.FC<SearchExampleTooltipProps> = ({
  onSelectExample,
  exampleQueries,
}) => {
  const examples = exampleQueries ?? [
    { id: "failed-requests", text: "Show failed requests today" },
    { id: "auth-errors", text: "auth errors in the last 3h" },
    { id: "api-calls", text: "API calls from a path that includes /api/v1/oz" },
  ];

  return (
    <TooltipProvider>
      <Tooltip delayDuration={150}>
        <TooltipTrigger asChild>
          <div data-testid="info-icon">
            <CircleInfoSparkle className="size-4 text-accent-9" />
          </div>
        </TooltipTrigger>
        <TooltipContent
          className="p-3 bg-gray-1 dark:bg-black drop-shadow-2xl border border-gray-6 rounded-lg text-accent-12 text-xs"
          data-testid="example-tooltip"
        >
          <div>
            <div className="font-medium mb-2 flex items-center gap-2 text-[13px]">
              <span>Try queries like:</span>
              <span className="text-[11px] text-gray-11">(click to use)</span>
            </div>
            <ul className="space-y-1.5 pl-1 [&_svg]:size-[10px] ">
              {examples.map((example) => (
                <li key={example.id} className="flex items-center gap-2">
                  <CaretRightOutline className="text-accent-9" />
                  <button
                    type="button"
                    className="hover:text-accent-11 transition-colors cursor-pointer hover:underline"
                    onClick={() => onSelectExample(example.text)}
                    data-testid={`example-${example.id}`}
                  >
                    "{example.text}"
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
