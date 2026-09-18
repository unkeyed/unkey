import { Button } from "~/components/ui/button";

export type ChartState =
  | { kind: "loading" }
  | { kind: "error"; message: string; onRetry?: () => void }
  | { kind: "empty" }
  | { kind: "populated" };

type Props = {
  state: ChartState;
  children: React.ReactNode;
};

export function ChartSlot({ state, children }: Props) {
  switch (state.kind) {
    case "loading":
      return (
        <div
          className="h-full w-full rounded-md bg-gray-3 motion-safe:animate-pulse"
          aria-busy="true"
        />
      );
    case "error":
      return (
        <div className="flex h-full flex-col items-center justify-center gap-3">
          <p className="text-gray-12 text-sm">{state.message}</p>
          {state.onRetry && (
            <Button variant="outline" onClick={state.onRetry}>
              Try again
            </Button>
          )}
        </div>
      );
    case "empty":
      return (
        <div className="flex h-full items-center justify-center text-gray-11 text-sm">
          No data for this time range
        </div>
      );
    case "populated":
      return children;
  }
}
