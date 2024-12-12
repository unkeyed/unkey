import { PropsWithChildren } from "react";

export function PageContent({ children }: PropsWithChildren) {
  return <div className="p-4 lg:p-8">{children}</div>;
}
