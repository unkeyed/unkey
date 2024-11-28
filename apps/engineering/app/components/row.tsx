import type { PropsWithChildren } from "react";

export const Row: React.FC<PropsWithChildren> = (props) => {
  return <div className="flex w-full items-center justify-around gap-8">{props.children}</div>;
};
