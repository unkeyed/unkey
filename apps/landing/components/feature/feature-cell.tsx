import { cn } from "@/lib/utils";
import * as React from "react";
import {
  ApiFirst,
  AutoKeyExpiration,
  DataRetention,
  DetectAndProtect,
  MultiCloud,
  RoleBasedAccess,
  SdkIcon,
  UsageLimits,
  VercelIntegration,
} from "./feature-svgs";

export enum iconNames {
  cloud = 0,
  data = 1,
  api = 2,
  role = 3,
  detect = 4,
  sdk = 5,
  vercel = 6,
  automatic = 7,
  usage = 8,
}
export interface IProps {
  iconName: iconNames;
}

const Feature = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("text-left", className)} {...props} />
  ),
);
Feature.displayName = "Feature";

const FeatureHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex flex-row space-y-1.5 p-6 ", className)} {...props} />
  ),
);
FeatureHeader.displayName = "FeatureHeader";

const FeatureTitle = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h3
    ref={ref}
    className={cn(
      "flex flex-row gap-4 text-base tracking-tight font-medium leading-6 text-white",
      className,
    )}
    {...props}
  />
));
FeatureTitle.displayName = "FeatureTitle";

const FeatureIcon = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { iconName: string }
>(({ className, iconName }, ref) => (
  <div ref={ref} className={cn("text-left pt-[9px] mr-4", className)}>
    {getSvg(iconName)}
  </div>
));
FeatureIcon.displayName = "FeatureIcon";

const FeatureContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "p-6 pt-0 text-[rgba(255,255,255,0.6)] font-normal text-sm leading-6",
        className,
      )}
      {...props}
    />
  ),
);
FeatureContent.displayName = "FeatureContent";

export { Feature, FeatureHeader, FeatureTitle, FeatureContent, FeatureIcon };

function getSvg(variant: string) {
  switch (variant) {
    case "cloud":
      return <MultiCloud />;
    case "data":
      return <DataRetention />;
    case "api":
      return <ApiFirst />;
    case "role":
      return <RoleBasedAccess />;
    case "detect":
      return <DetectAndProtect />;
    case "sdk":
      return <SdkIcon />;
    case "vercel":
      return <VercelIntegration />;
    case "automatic":
      return <AutoKeyExpiration />;
    case "usage":
      return <UsageLimits />;
    default:
      return <MultiCloud />;
  }
}
