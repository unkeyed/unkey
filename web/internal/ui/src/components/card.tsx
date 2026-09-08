import type * as React from "react";

import { cn } from "../lib/utils";

function Card({
  className,
  ref,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { ref?: React.Ref<HTMLDivElement> }) {
  return (
    <div
      ref={ref}
      className={cn("w-full rounded-lg border border-gray-4 bg-background", className)}
      {...props}
    />
  );
}

function CardHeader({
  className,
  ref,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { ref?: React.Ref<HTMLDivElement> }) {
  return (
    <div ref={ref} className={cn("flex flex-col space-y-1.5 p-6 pb-0", className)} {...props} />
  );
}

function CardTitle({
  className,
  ref,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement> & { ref?: React.Ref<HTMLParagraphElement> }) {
  return (
    <h2
      ref={ref}
      className={cn("text-2xl font-semibold leading-none tracking-tight pb-1", className)}
      {...props}
    />
  );
}

function CardDescription({
  className,
  ref,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement> & { ref?: React.Ref<HTMLParagraphElement> }) {
  return <p ref={ref} className={cn("text-sm text-content-subtle", className)} {...props} />;
}

function CardContent({
  className,
  ref,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { ref?: React.Ref<HTMLDivElement> }) {
  return <div ref={ref} className={cn("p-6", className)} {...props} />;
}

function CardFooter({
  className,
  ref,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { ref?: React.Ref<HTMLDivElement> }) {
  return (
    <div
      ref={ref}
      className={cn("flex items-center px-6 py-3 mt-3 border-t border-gray-4", className)}
      {...props}
    />
  );
}

export { Card, CardHeader, CardFooter, CardTitle, CardDescription, CardContent };
