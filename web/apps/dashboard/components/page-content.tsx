import type { PropsWithChildren } from "react";

export function PageContent({ children }: PropsWithChildren) {
  return <div className="p-4">{children}</div>;
}
