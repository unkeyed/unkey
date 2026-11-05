import { Mysql, Postgres } from "@unkey/icons";
import { cn } from "cn";
import type { ComponentType } from "react";
import type { Engine } from "./database-model";

function PostgresIcon({ className }: { className?: string }) {
  return <Postgres className={cn("text-info-11", className)} />;
}

export const engineIcon: Record<Engine, ComponentType<{ className?: string }>> = {
  postgres: PostgresIcon,
  mysql: Mysql,
};

export const TILE_SIZE = {
  sm: { tile: "size-6 rounded-md", icon: "size-3" },
  md: { tile: "size-7 rounded-lg", icon: "size-4" },
} as const;

export function EngineTile({ engine, size }: { engine: Engine; size: keyof typeof TILE_SIZE }) {
  const Icon = engineIcon[engine];
  return (
    <span
      className={cn(
        "flex shrink-0 items-center justify-center border bg-raised",
        TILE_SIZE[size].tile,
      )}
    >
      <Icon className={TILE_SIZE[size].icon} />
    </span>
  );
}
