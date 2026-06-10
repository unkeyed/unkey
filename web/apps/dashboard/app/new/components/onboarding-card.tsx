import { cn } from "@/lib/utils";
import type React from "react";

export function OnboardingCard({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("border border-gray-5 rounded-2xl w-full p-8", className)} {...props} />
  );
}

export function OnboardingCardHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col", className)} {...props} />;
}

export function OnboardingCardTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2 className={cn("text-gray-12 font-semibold text-lg leading-8", className)} {...props} />
  );
}

export function OnboardingCardDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p className={cn("text-gray-11 font-normal text-[13px] leading-6", className)} {...props} />
  );
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
  return <div className={cn("mt-8 border-t border-gray-5 pt-8", className)} {...props} />;
}
