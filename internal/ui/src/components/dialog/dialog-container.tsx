"use client";
import type { PropsWithChildren, ReactNode } from "react";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import { cn } from "../../lib/utils";
import { Dialog, DialogContent } from "./components/dialog";
import { DefaultDialogContentArea, DefaultDialogFooter, DefaultDialogHeader } from "./components/dialog-parts";

type DialogContainerProps = PropsWithChildren<{
    className?: string;
    isOpen: boolean;
    onOpenChange: (value: boolean) => void;
    title: string;
    footer?: ReactNode;
    contentClassName?: string;
    preventAutoFocus?: boolean;
    subTitle?: string;
}>;

export const DialogContainer = ({
    className,
    isOpen,
    subTitle,
    onOpenChange,
    title,
    children,
    footer,
    contentClassName,
    preventAutoFocus = true,
}: DialogContainerProps) => {
    return (
        <Dialog open={isOpen} onOpenChange={onOpenChange}>
            <DialogContent
                className={cn(
                    "drop-shadow-2xl border-gray-4 overflow-hidden !rounded-2xl p-0 gap-0",
                    className,
                )}
                onOpenAutoFocus={(e: Event) => {
                    if (preventAutoFocus) {
                        e.preventDefault();
                    }
                }}
            >
                <DefaultDialogHeader title={title} subTitle={subTitle} />
                <DefaultDialogContentArea className={contentClassName}>{children}</DefaultDialogContentArea>
                {footer && <DefaultDialogFooter>{footer}</DefaultDialogFooter>}
            </DialogContent>
        </Dialog>
    );
};

export { DefaultDialogHeader, DefaultDialogContentArea, DefaultDialogFooter };