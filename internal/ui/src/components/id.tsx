"use client";
import { Slot } from "@radix-ui/react-slot";
import * as React from "react";
import { cn } from "../lib/utils";
import { useState } from "react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./ui/tooltip";
import { CopyIcon } from "lucide-react";



type copyProps = React.SVGProps<SVGSVGElement> & {
    width?: number;
    height?: number;
};


type IdProps = {
    value?: string;
    truncate?: number;
    className?: string;
};

export const Id: React.FC<IdProps> = ({ className, value, truncate, ...props }) => {
    const [isHover, setIsHover] = useState(false);
    const [iconHover, setIconHover] = useState(false);
    const copyTextToClipboard = async (value: string) => {
        try {
            await navigator.clipboard.writeText(value ?? "");
        } catch (error) {
            console.error('Failed to copy: ', error);
        }
    };

    function generateString() {
        const char = '•';
        let length = 0;

        if (!value?.length || !truncate) {
            return '';
        } else {
            length = value?.length - truncate;
        }
        return Array.from({ length }, () => char).join('');
    }

    const truncateValue = truncate ? value?.slice(0, truncate) + generateString() : value;
    const Comp = "div";

    return (
        <Comp
            className={cn("inline-flex group items-center transition duration-150 justify-center gap-3 whitespace-nowrap tracking-normal rounded-lg  font-medium transition-colors disabled:pointer-events-none focus:outline-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 bg-base-12 w-fit max-w-96 border border-accent-6 text-accent-12 radius radius-2 font-mono font-normal h-8 px-3 py-1 text-xs", className)}
            onClick={() => copyTextToClipboard(value ?? "Copied")}
            {...props}
            onMouseEnter={() => setIsHover(true)}
            onFocus={() => setIsHover(true)}
            onMouseLeave={() => setIsHover(false)}
            onBlur={() => setIsHover(false)}
        >
            {truncateValue}
            {isHover &&
                <TooltipProvider>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <div onClick={() => copyTextToClipboard(value ?? "")}
                                onMouseEnter={() => setIconHover(true)}
                                onFocus={() => setIconHover(true)}
                                onMouseLeave={() => setIconHover(false)}
                                onBlur={() => setIconHover(false)} ><CopyIcon/></div>
                        </TooltipTrigger>
                        <TooltipContent className="bg-white h-8" side="bottom">
                            Copy ID
                        </TooltipContent>
                    </Tooltip>
                </TooltipProvider>}
        </Comp>);
};
Id.displayName = "Id";


