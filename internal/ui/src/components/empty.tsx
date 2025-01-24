// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import { cn } from "../lib/utils";

interface EmptyRootProps extends React.HTMLAttributes<HTMLDivElement> {}
export function Empty({ className, children, ...props }: EmptyRootProps) {
  return (
    <div
      className={cn(
        "flex flex-col p-8 text-center h-full w-full items-center justify-center",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
}

type EmptyIconProps = React.HTMLAttributes<HTMLDivElement>;

Empty.Icon = function EmptyIcon({ className }: EmptyIconProps) {
  return (
    <div className={cn("flex h-28 w-28 justify-center items-center", className)}>
      <div className="relative z-10">
        <div className="absolute top-0 left-1/2 -translate-x-1/2 bg-gradient-to-r from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-32 h-[1px] " />
        <div className="absolute bottom-0 left-1/2 -translate-x-1/2 bg-gradient-to-r from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-32 h-[1px]  " />
        <div className="absolute left-0 top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-[1px]" />
        <div className="absolute right-0 top-1/2 -translate-y-1/2 h-32 bg-gradient-to-t from-[hsla(240,100%,17%,0.01)] via-[hsla(240,100%,10%,0.06)] to-[hsla(240,100%,17%,0.01)] dark:from-[hsla(0,0%,0%,0)] dark:via-[hsla(211,66%,92%,0.3)] dark:to-[hsla(0,0%,0%,0)] w-[1px] " />
        <div className="flex w-16 h-16 items-center justify-center rounded-2xl bg-gray-2 border border-[1px] border-[hsla(240,100%,10%,0.06)] dark:border-[hsla(211,66%,92%,0.2)] text-accent-12 z-50 [&_svg]:pointer-events-none [&_svg]:size-7 [&_svg]:shrink-0">
          <svg viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
            <g fill="currentColor">
              <circle cx="14.75" cy="1.75" fill="currentColor" r=".75" stroke="none" />
              <path
                d="M3.869,1.894l-.947-.315-.315-.947c-.103-.306-.609-.306-.712,0l-.315,.947-.947,.315c-.153,.051-.256,.194-.256,.356s.104,.305,.256,.356l.947,.315,.315,.947c.051,.153,.194,.256,.356,.256s.305-.104,.356-.256l.315-.947,.947-.315c.153-.051,.256-.194,.256-.356s-.104-.305-.256-.356Z"
                fill="currentColor"
                stroke="none"
              />
              <path
                d="M5.223,5.526c-.012-.115-.015-.216-.015-.334,0-1.887,1.53-3.417,3.417-3.417,1.575,0,2.901,1.066,3.297,2.516"
                fill="none"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1"
              />
              <path
                d="M6.865,8.894c-2.701,.164-4.701-.232-4.844-1.07-.187-1.094,2.861-2.527,6.808-3.201,3.947-.674,7.298-.334,7.485,.76,.151,.886-1.822,1.995-4.676,2.743"
                fill="none"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1"
              />
              <line
                fill="none"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1"
                x1="7.006"
                x2="6"
                y1="7.689"
                y2="16.25"
              />
              <line
                fill="none"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1"
                x1="11"
                x2="16.25"
                y1="7"
                y2="16.25"
              />
              <ellipse
                cx="9.002"
                cy="7.34"
                fill="currentColor"
                rx="2.026"
                ry=".316"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1"
                transform="translate(-1.13 1.66) rotate(-9.918)"
              />
            </g>
          </svg>
        </div>
      </div>
    </div>
  );
};

type EmptyTitleProps = React.HTMLAttributes<HTMLHeadingElement>;

Empty.Title = function EmptyTitle({ className, ...props }: EmptyTitleProps) {
  return (
    <h2
      className={cn("text-accent-12 mt-3 font-semibold text-[15px] leading-6", className)}
      {...props}
    />
  );
};

type EmptyDescriptionProps = React.HTMLAttributes<HTMLParagraphElement>;

Empty.Description = function EmptyDescription({ className, ...props }: EmptyDescriptionProps) {
  return (
    <p
      className={cn("text-accent-11 text-center text-xs font-normal leading-6 mt-1", className)}
      {...props}
    />
  );
};

type EmptyActionsProps = React.HTMLAttributes<HTMLDivElement>;

Empty.Actions = function EmptyActions({ className, children, ...props }: EmptyActionsProps) {
  return (
    <div className={cn("w-full flex items-center justify-center gap-4 mt-2", className)} {...props}>
      {children}
    </div>
  );
};
