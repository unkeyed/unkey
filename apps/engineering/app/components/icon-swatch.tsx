import type { PropsWithChildren } from "react";

type Props = {
  name: string;
};
export const Icon: React.FC<PropsWithChildren<Props>> = (props) => {
  return (
    <div className="flex flex-col items-center justify-center gap-4 text-gray-12">
      <div className="flex items-center justify-center border rounded-lg size-12 aspect-square border-gray-5 bg-gray-3 ">
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
