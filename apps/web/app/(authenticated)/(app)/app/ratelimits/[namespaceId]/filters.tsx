"use client";

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
import { useSearchParams } from "next/navigation";
import { usePathname } from "next/navigation";
import { parseAsStringEnum, useQueryState } from "nuqs";
import React, { useCallback, useTransition } from "react";

export const interval = {
  "60m": "Last 60 minutes",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
  "90d": "Last 3 months",
} as const;

export type Interval = keyof typeof interval;

export const Filters: React.FC = () => {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const [selected, setSelected] = useQueryState(
    "interval",
    parseAsStringEnum(["60m", "24h", "7d", "30d", "90d"]).withDefault("7d").withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
    }),
  );

  return (
    <div className="flex items-center gap-2">
      <div>
        <Select
          value={selected}
          onValueChange={(i: Interval) => {
            setSelected(i);
            startTransition(() => {});
          }}
        >
          <SelectTrigger>
            <SelectValue defaultValue={selected} />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(interval).map(([id, label]) => (
              <SelectItem key={id} value={id}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

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
  );
};

/**
 * Utility hook to modify the search params of the current URL
 */
export function useModifySearchParams() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams()!;

  const hrefWithSearchparam = useCallback(
    (name: string, value: string) => {
      const params = new URLSearchParams(searchParams.toString());
      params.set(name, value);
      return `${pathname}?${params.toString()}`;
    },
    [pathname, searchParams],
  );

  return {
    set: (key: string, value: string) => {
      router.push(hrefWithSearchparam(key, value), { scroll: false });
    },
  };
}
