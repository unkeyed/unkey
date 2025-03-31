import type { PropsWithChildren } from "react";

export const EmptyComponentSpacer = ({ children }: PropsWithChildren) => {
  return (
    <div className="h-full min-h-[300px] flex items-center justify-center">
      <div className="flex justify-center items-center">{children}</div>
    </div>
  );
};
