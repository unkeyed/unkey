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
    <div ref={ref} className={cn("text-left flex flex-col gap-4", className)} {...props} />
  ),
);
Feature.displayName = "Feature";

const FeatureHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex items-center gap-2", className)} {...props} />
  ),
);
FeatureHeader.displayName = "FeatureHeader";

const FeatureTitle = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h3
    ref={ref}
    className={cn("text-base tracking-tight font-medium leading-6 text-white", className)}
    {...props}
  />
));
FeatureTitle.displayName = "FeatureTitle";

const FeatureIcon = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { iconName: string }
>(({ className, iconName }, ref) => (
  <div ref={ref} className={className}>
    {getSvg(iconName)}
  </div>
));
FeatureIcon.displayName = "FeatureIcon";

const FeatureContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("text-white/60 font-normal text-sm leading-6", className)}
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
