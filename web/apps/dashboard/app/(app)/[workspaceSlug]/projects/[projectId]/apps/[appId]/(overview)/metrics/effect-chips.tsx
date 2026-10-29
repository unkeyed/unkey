import { match } from "@unkey/match";
import { Badge, type BadgeProps } from "@unkey/ui";
import { cn } from "cn";
import type { DeployEffect, EffectChange, EffectTone } from "./deploy-effects";

const TONE_VARIANT: Record<EffectTone, BadgeProps["variant"]> = {
  worse: "error",
  better: "success",
  neutral: "primary",
};

export function EffectChips({
  effects,
  className,
}: {
  effects: DeployEffect[];
  className?: string;
}) {
  return (
    <div className={cn("flex flex-wrap gap-1.5", className)}>
      {effects.map((effect) => (
        <Badge
          key={effect.metric}
          variant={TONE_VARIANT[effect.tone]}
          size="sm"
          className="font-mono text-2xs tabular-nums"
        >
          {effect.label} {changeLabel(effect.change)}
        </Badge>
      ))}
    </div>
  );
}

function changeLabel(change: EffectChange): string {
  return match(change)
    .with({ kind: "from-zero" }, () => "new")
    .with({ kind: "percent" }, ({ percent }) => {
      const rounded = Math.round(percent);
      return `${rounded > 0 ? "+" : ""}${rounded}%`;
    })
    .exhaustive();
}
