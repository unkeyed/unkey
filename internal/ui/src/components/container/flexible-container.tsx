// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../../lib/utils";
import { type VariantProps, cva } from "class-variance-authority";

const flexibleContainerVariants = cva(
    "flex flex-col", // Base padding and centering
    {
        variants: {
            width: {
                full: "w-full",
                xlarge: "w-full max-w-xl", // 36rem 576px
                large: "w-full max-w-lg", // 32rem 512px
                medium: "w-full max-w-md", // 28rem 448px
                small: "w-full max-w-sm", // 24rem 384px
                xsmall: "w-full max-w-xs", // 20rem 320px
            },
            padding: {
                none: "p-0 m-0",
                small: "px-4 py-6 lg:px-6 lg:py-8",
                medium: "px-6 py-8 lg:px-8 lg:py-10",
                large: "px-8 py-10 lg:px-10 lg:py-12",
            },
            horizontalPosition: {
                center: "self-center",
                left: "self-start",
                right: "self-end",
            },
            verticalPosition: {
                center: "justify-center",
                top: "justify-start",
                bottom: "justify-end",
            },
            border: {
                none: "border-none",
                top: "border-t border-gray-4",
                bottom: "border-b border-gray-4",
                left: "border-l border-gray-4",
                right: "border-r border-gray-4",
                all: "border border-gray-4",
            },
            borderRadius: {
                none: "rounded-none",
                small: "rounded-sm",
                medium: "rounded-md",
                large: "rounded-lg",
                xlarge: "rounded-xl",
                xxlarge: "rounded-2xl",
                xxxlarge: "rounded-3xl",
                
            },
        },
        defaultVariants: {
            width: "full",
            horizontalPosition: "center",
            verticalPosition: "center",
            padding: "none",
            border: "none",
            borderRadius: "none",
        },
    }
);
type FlexibleContainerProps = React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof flexibleContainerVariants>;

export const FlexibleContainer = ({
    className,
    width,
    children,
    verticalPosition,
    horizontalPosition,
    padding,
    border,
    borderRadius,
}: FlexibleContainerProps) => {


    return (
        <div
            className={cn(flexibleContainerVariants({ width, verticalPosition, horizontalPosition, padding, border, borderRadius, className}))}
        >
            {children}
        </div>
    );
}

FlexibleContainer.displayName = "FlexibleContainer";