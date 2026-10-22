import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { IconCodeOutline12, IconLockOutline12 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "cn";

export type EnvVarType = EnvVar["type"];

export const TYPE_MARKERS = {
  writeonly: {
    Icon: IconLockOutline12,
    name: "Sensitive",
    label: "Sensitive. You can't view the value.",
  },
  recoverable: {
    Icon: IconCodeOutline12,
    name: "Not sensitive",
    label: "Not sensitive. You can view the value.",
  },
} satisfies Record<EnvVarType, { Icon: typeof IconLockOutline12; name: string; label: string }>;

export function TypeMarker({ type, className }: { type: EnvVarType; className?: string }) {
  const { Icon, label } = TYPE_MARKERS[type];

  return (
    <InfoTooltip content={label} position={{ side: "top" }} asChild>
      <span
        className={cn(
          "relative flex size-6 items-center justify-center rounded-md border bg-raised text-gray-12 transition-opacity",
          className,
        )}
      >
        <Icon />
        <span className="sr-only">{label}</span>
      </span>
    </InfoTooltip>
  );
}
