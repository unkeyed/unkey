"use client";
import { TaskChecked, TaskUnchecked } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "./tooltip";

type IdProps = {
  value: string;
  truncate?: number;
  className?: string;
};

export const Id: React.FC<IdProps> = ({ className, value, truncate, ...props }) => {
  const [isCopied, setIsCopied] = React.useState(false);
  const copyTextToClipboard = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value ?? "");
      setIsCopied(true);
    } catch (error) {
      console.error("Failed to copy: ", error);
    }
  };

  React.useEffect(() => {
    if (!isCopied) {
      return;
    }
    const timer = setTimeout(() => {
      setIsCopied(false);
    }, 2000);
    return () => clearTimeout(timer);
  }, [isCopied]);

  const ellipse = "••••";
  const truncateValue = truncate ? value?.slice(0, truncate) + ellipse : value;
  const Comp = "button";

  return (
    <Comp
      className={cn(
        "relative inline-flex w-full ring-2 ring-transparent focus:ring-gray-6 group items-center transition duration-150 justify-center gap-3 whitespace-nowrap tracking-normal rounded-lg font-medium bg-gray-1 w-fit max-w-96 border border-accent-6 hover:border-accent-8 text-accent-12 radius radius-2 font-mono h-8 px-3 py-1 text-xs overflow-hidden",
        className,
      )}
      onClick={() => copyTextToClipboard(value ?? "")}
      {...props}
    >
      {truncateValue}
      <Tooltip>
        <div className="absolute flex h-8 w-8 top-0 right-0 invisible group-hover:visible group-focus:visible">
          <TooltipTrigger asChild>
            <div className=" flex justify-end border w-full border-none h-full text-gray-1 bg-gray-1">
              {!isCopied ? (
                <TaskUnchecked className="size=[12] text-gray-10 item-end my-auto mr-2 bg-base-1" />
              ) : (
                <TaskChecked className="size=[12] text-gray-10 item-end my-auto mr-2" />
              )}
            </div>
          </TooltipTrigger>
          <TooltipContent side="bottom">Copy ID</TooltipContent>
        </div>
      </Tooltip>
    </Comp>
  );
};
Id.displayName = "Id";
