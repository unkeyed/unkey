import { IconTriangleWarningOutline12 } from "@unkey/icons";
import { cn } from "cn";
import type { ReactNode } from "react";

type AppBannerTone = "error" | "warning";

const TONE_CLASS: Record<AppBannerTone, string> = {
  error: "bg-error-9 text-white",
  warning: "bg-warning-9 text-black",
};

export function AppBanner({
  tone,
  action,
  children,
}: {
  tone: AppBannerTone;
  action: ReactNode;
  children: ReactNode;
}) {
  return (
    <div
      data-app-banner
      className={cn(
        "flex h-(--app-banner-h) w-full shrink-0 items-center justify-center gap-2 overflow-hidden px-4 text-center font-medium text-sm",
        TONE_CLASS[tone],
      )}
    >
      <IconTriangleWarningOutline12 className="shrink-0" />
      <span className="min-w-0 truncate">{children}</span>
      <span className="shrink-0 *:underline *:underline-offset-2 *:hover:opacity-80 *:focus-visible:outline-2 *:focus-visible:outline-offset-2 *:focus-visible:outline-current *:disabled:opacity-60">
        {action}
      </span>
    </div>
  );
}
