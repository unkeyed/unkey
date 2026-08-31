import type { ReactNode } from "react";

export function DestinationStepContainer({ children }: { children: ReactNode }) {
  return <div className="flex w-[600px] max-w-[calc(100vw-2rem)] flex-col gap-3">{children}</div>;
}
