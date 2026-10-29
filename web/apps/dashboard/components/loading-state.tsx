import { Loading } from "@unkey/ui";
import type { ReactNode } from "react";

type LoadingStateProps = {
  message?: string;
  children?: ReactNode;
};

export function LoadingState({ message = "Loading...", children }: LoadingStateProps) {
  return (
    <div className="flex-1 relative flex flex-col overflow-hidden bg-background lg:flex-row">
      <div className="flex items-center justify-center w-full flex-1">
        <div className="flex flex-col items-center gap-4">
          <Loading size={24} />
          <output className="text-sm text-gray-11">{message}</output>
          {children}
        </div>
      </div>
    </div>
  );
}
