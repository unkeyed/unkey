import type { PropsWithChildren } from "react";

type Props = {
  name: string;
};
export const Icon: React.FC<PropsWithChildren<Props>> = (props) => {
  return (
    <div className="flex flex-col justify-center items-center text-gray-12 gap-4">
      <div className="size-12 flex items-center  justify-center aspect-square border border-gray-5 rounded-lg bg-gray-3 ">
        {props.children}
      </div>
      <span className="text-sm">
        {"<"}
        {props.name}
        {"/>"}
      </span>
    </div>
  );
};

Icon.displayName = "Icon";
