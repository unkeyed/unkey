"use client";
import * as React from "react";
import { cn } from "../lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "./tooltip";
import { CopyIcon } from "lucide-react";


type IdProps = {
    value: string;
    truncate?: number;
    className?: string;
};

export const Id: React.FC<IdProps> = ({ className, value, truncate, ...props }) => {
    const copyTextToClipboard = async (value: string) => {
        try {
            await navigator.clipboard.writeText(value ?? "");
        } catch (error) {
            console.error('Failed to copy: ', error);
        }
    };

    const ellipse = '••••';
    const truncateValue = truncate ? value?.slice(0, truncate) + ellipse : value;
    const Comp = "button";

    return (
        <Comp
            className={cn("inline-flex w-full ring-2 ring-transparent focus:ring-gray-6 group items-center transition duration-150 justify-center gap-3 whitespace-nowrap tracking-normal rounded-lg font-medium bg-base-12 w-fit max-w-96 border border-accent-6 hover:border-accent-8 text-accent-12 radius radius-2 font-mono h-8 px-3 py-1 text-xs", className)}
            onClick={() => copyTextToClipboard(value ?? "")}
            {...props}
        >
            {truncateValue}
                <Tooltip>
                <div className="invisible group-hover:visible group-focus:visible">
                    <TooltipTrigger asChild className=" ">
                        <CopyIcon className=" text-gray-9 item-center" size={10.5} strokeWidth={3} />
                    </TooltipTrigger>
                    <TooltipContent side="bottom">
                        Copy ID
                    </TooltipContent>
                </div>
                </Tooltip>
        </Comp>);
};
Id.displayName = "Id";


