"use client";
import { TaskChecked, TaskUnchecked } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";
import { InfoTooltip } from "./info-tooltip";

type IdProps = {
  /**
   * Value displayed on the component.
   */
  value: string;
  /**
   * Number to truncate the value to. If the value is longer than the truncate number, it will be truncated and an ellipse will be added to the end.
   */
  truncate?: number;
  /**
   * Any additional classes to apply to the component.
   */
  className?: string;
};

export const Id: React.FC<IdProps> = ({ className, value, truncate, ...props }) => {
  const [isCopied, setIsCopied] = React.useState(false);
  const copyTextToClipboard = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
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

  return (
    <button
      type="button"
      className={cn(
        "relative inline-flex ring-2 ring-transparent no-underline focus:ring-gray-6 group items-center transition duration-150 justify-center gap-3 whitespace-nowrap tracking-normal rounded-lg font-medium bg-gray-1 w-fit max-w-96 border border-accent-6 hover:border-accent-8 text-gray-12 font-mono h-8 px-3 py-1 text-xs overflow-hidden",
        className,
      )}
      onClick={() => copyTextToClipboard(value)}
      aria-label={`Copy ID: ${value}`}
      {...props}
    >
      {truncateValue}
      <InfoTooltip position={{ side: "bottom" }} content={value}>
        <div className=" flex justify-end border w-full border-none h-full bg-accent-1">
          {isCopied ? (
            <TaskChecked className="item-end my-auto mr-2 bg-gray-1" />
          ) : (
            <TaskUnchecked className="item-end my-auto mr-2 bg-gray-1" />
          )}
        </div>
      </InfoTooltip>
    </button>
  );
};
Id.displayName = "Id";
