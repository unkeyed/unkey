"use client";

import { ArrayInput } from "@/components/array-input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { RefreshCw } from "lucide-react";
import { useRouter } from "next/navigation";
import { parseAsArrayOf, parseAsString, parseAsStringEnum, useQueryState } from "nuqs";
import React, { useTransition } from "react";

export const intervals = {
  "60m": "Last 60 minutes",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
  "90d": "Last 3 months",
} as const;

export type Interval = keyof typeof intervals;

export const Filters: React.FC<{ identifier?: boolean; interval?: boolean }> = (props) => {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const [interval, setInterval] = useQueryState(
    "interval",
    parseAsStringEnum(["60m", "24h", "7d", "30d", "90d"]).withDefault("7d").withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
    }),
  );
  const [identifier, setIdentifier] = useQueryState(
    "identifier",
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
    }),
  );

  return (
    <div className="flex items-center justify-between gap-2">
      {props.identifier ? (
        <div>
          <ArrayInput
            title="Identifiers"
            selected={identifier}
            setSelected={(v) => {
              setIdentifier(v);
              startTransition(() => {});
            }}
          />
        </div>
      ) : null}{" "}
      {props.interval ? (
        <div>
          <Select
            value={interval}
            onValueChange={(i: Interval) => {
              setInterval(i);
              startTransition(() => {});
            }}
          >
            <SelectTrigger>
              <SelectValue defaultValue={interval} />
            </SelectTrigger>
            <SelectContent>
              {Object.entries(intervals).map(([id, label]) => (
                <SelectItem key={id} value={id}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}
      <div>
        <Button
          size="icon"
          variant="secondary"
          onClick={() => {
            startTransition(router.refresh);
          }}
        >
          <RefreshCw className={cn("w-4 h-4", { "animate-spin": isPending })} />
        </Button>
      </div>
    </div>
  );
};
