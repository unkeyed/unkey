import { cn } from "@/lib/utils";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import {
  AlertAlertIcon,
  AlertErrorIcon,
  AlertInfoIcon,
  AlertSuccessIcon,
  AlertWarningIcon,
} from "./alert-svgs";

const alertVariants = cva("relative rounded-2xl p-6 text-left", {
  variants: {
    variant: {
      info: "text-[#3DC5FA] bg-[linear-gradient(90deg,rgba(61,197,250,0.15)_0%,rgba(61,197,250,0.1)_100%),radial-gradient(100%_608.33%_at_0%_50%,rgba(61,197,250,0.15)_0%,rgba(61,197,250,0.05)_100%)]",
      success:
        "text-[#3CEEAE] bg-[linear-gradient(90deg,rgba(60,238,174,0.15)_0%,rgba(60,238,174,0.1)_100%),radial-gradient(100%_608.33%_at_0%_50%,rgba(60,238,174,0.15)_0%,rgba(60,238,174,0.05)_100%)]",
      alert:
        "text-[#9D72FF] bg-[linear-gradient(90deg,rgba(157,114,255,0.15)_0%,rgba(157,114,255,0.1)_100%),radial-gradient(100%_608.33%_at_0%_50%,rgba(157,114,255,0.15)_0%,rgba(157,114,255,0.05)_100%)]",
      warning:
        "text-[#FFD55D] bg-[linear-gradient(90deg,rgba(255,213,93,0.15)_0%,rgba(255,213,93,0.1)_100%),radial-gradient(100%_608.33%_at_0%_50%,rgba(255,213,93,0.15)_0%,rgba(255,213,93,0.05)_100%)]",
      error:
        "text-[#FB1048] bg-[linear-gradient(90deg,rgba(251,16,72,0.15)_0%,rgba(251,16,72,0.1)_100%),radial-gradient(100%_608.33%_at_0%_50%,rgba(251,16,72,0.15)_0%,rgba(251,16,72,0.05)_100%)]",
    },
    backgroundVariants: {
      info: "bg-[linear-gradient(90deg,rgba(61,197,250,0.15)_0%,rgba(61,197,250,0.1)_100%)] p-[.75px]",
      success:
        "bg-[linear-gradient(90deg,rgba(60,238,174,0.15)_0%,rgba(60,238,174,0.1)_100%)] p-[.75px]",
      alert:
        "bg-[linear-gradient(90deg,rgba(157,114,255,0.15)_0%,rgba(157,114,255,0.1)_100%)] p-[.75px]",
      warning:
        "bg-[linear-gradient(90deg,rgba(255,213,93,0.15)_0%,rgba(255,213,93,0.1)_100%)] p-[.75px]",
      error:
        "bg-[linear-gradient(90deg,rgba(251,16,72,0.15)_0%,rgba(251,16,72,0.1)_100%)] p-[.75px]",
    },
  },

  defaultVariants: {
    variant: "info",
  },
});
// TODO: Colors are still to light. Not sure if gradient is correct.
// TODO: Border is not showing correctly
const Alert = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof alertVariants>
>(({ className, variant, ...props }, ref) => (
  <div className={cn(className)}>
    <div className={cn(alertVariants({ backgroundVariants: variant }))}>
      <div ref={ref} role="alert" className={cn(alertVariants({ variant }))} {...props} />
    </div>
  </div>
));
Alert.displayName = "Alert";

const AlertDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement> & VariantProps<typeof alertVariants>
>(({ className, variant, ...props }, ref) => (
  <div className="flex flex-row gap-4">
    <div className="pr-3 pt-2">{variant ? getSvg(variant) : null}</div>
    <div ref={ref} className={cn("my-auto", className)} {...props} />
  </div>
));
AlertDescription.displayName = "AlertDescription";

export { Alert, AlertDescription };

function getSvg(variant: string) {
  switch (variant) {
    case "info":
      return <AlertInfoIcon />;
    case "success":
      return <AlertSuccessIcon />;
    case "alert":
      return <AlertAlertIcon />;
    case "warning":
      return <AlertWarningIcon />;
    case "error":
      return <AlertErrorIcon />;
    default:
      return <AlertInfoIcon />;
  }
}
