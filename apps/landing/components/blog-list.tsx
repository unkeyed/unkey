import { cn } from "@/lib/utils";
import React from "react";

type BlogListProps = {
  children?: React.ReactNode;
  className?: string;
};

export function BlogList({ children, className }: BlogListProps) {
  // console.log("BlogList children", children);
  return (
    <ul className={cn("flex flex-col pl-28 list-disc text-white gap-6", className)}>{children}</ul>
  );
}
export function BlogListNumbered({ children, className }: BlogListProps) {
  // console.log("BlogList children", children);
  return (
    <ol className={cn("flex flex-col pl-28 list-decimal text-white gap-6 ", className)}>
      {children}
    </ol>
  );
}
export function BlogListItem({ children, className }: BlogListProps) {
  return (
    <li
      className={cn(
        "pl-0 leading-8 font-normal text-xl text-white/60",

        className,
      )}
    >
      <span className="text-lg">{children}</span>
    </li>
  );
}
