import { cn } from "@/lib/utils";
import type React from "react";

type BlogListProps = {
  children?: React.ReactNode;
  className?: string;
};

export function BlogList({ children, className }: BlogListProps) {
  // console.log("BlogList children", children);
  return (
    <ul className={cn("flex flex-col list-disc pl-6 text-white gap-4", className)}>{children}</ul>
  );
}
export function BlogListNumbered({ children, className }: BlogListProps) {
  // console.log("BlogList children", children);
  return (
    <ol className={cn("flex flex-col list-decimal pl-6 text-white gap-6 ", className)}>
      {children}
    </ol>
  );
}
export function BlogListItem({ children, className }: BlogListProps) {
  return (
    <li
      className={cn(
        "pl-6 leading-8 font-normal sm:text-lg text-white/60",

        className,
      )}
    >
      <span className="text-lg">{children}</span>
    </li>
  );
}
