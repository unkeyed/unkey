import { Check, ChevronDown, FlaskConical } from "lucide-react";
import { Button } from "~/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "~/components/ui/dropdown-menu";

export type DemoState = "empty" | "p50" | "p99" | "max";

const STATES: Record<DemoState, { label: string; description: string }> = {
  empty: { label: "Empty", description: "0 keys" },
  p50: { label: "p50", description: "3 keys · median" },
  p99: { label: "p99", description: "7 keys · top 1%" },
  max: { label: "Max", description: "37,566 keys · largest identity" },
};

type Props = {
  value: DemoState;
  onSelect: (next: DemoState) => void;
};

export function PreviewSwitcher({ value, onSelect }: Props) {
  return (
    <div className="fixed right-4 bottom-4 z-50">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="shadow-md">
            <FlaskConical />
            Preview: {STATES[value].label}
            <ChevronDown />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" side="top" className="min-w-48">
          <DropdownMenuLabel>Preview state</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {(Object.entries(STATES) as [DemoState, (typeof STATES)[DemoState]][]).map(
            ([state, { label, description }]) => (
              <DropdownMenuItem
                key={state}
                onSelect={(e) => {
                  e.preventDefault();
                  onSelect(state);
                }}
                className="justify-between gap-4"
              >
                <div className="flex flex-col">
                  <span>{label}</span>
                  <span className="text-gray-10 text-xs">{description}</span>
                </div>
                {value === state && <Check className="size-3.5 text-gray-11" />}
              </DropdownMenuItem>
            ),
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
