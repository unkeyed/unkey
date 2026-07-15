import { cn } from "@/lib/utils";
import { Card } from "@unkey/ui";
import type React from "react";

export function OnboardingCard({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <Card
      className={cn(
        "w-full max-w-[440px] rounded-xl border-gray-5 bg-gray-1 px-6 py-10 sm:px-12 shadow-xs",
        className,
      )}
      {...props}
    />
  );
}

export function OnboardingCardHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col gap-2", className)} {...props} />;
}

export function OnboardingCardTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2
      className={cn("text-2xl font-semibold tracking-tight text-gray-12", className)}
      {...props}
    />
  );
}

export function OnboardingCardDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("text-sm text-gray-11", className)} {...props} />;
}

export function OnboardingCardContent({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mt-4", className)} {...props} />;
}

export function OnboardingCardFooter({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mt-8", className)} {...props} />;
}
