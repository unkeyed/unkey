"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useRouter } from "next/navigation";
import { useSearchParams } from "next/navigation";
import { usePathname } from "next/navigation";
import type React from "react";
import { useCallback, useState } from "react";

export const interval = {
  "24h": "Last 24 Hours",
  "7d": "Last 7 Days",
  "30d": "Last 30 Days",
  "90d": "Last 3 Months",
} as const;

export type Interval = keyof typeof interval;

export type Props = {
  defaultSelected: Interval;
  className?: string;
  onChange?: (i: string) => void;
};

export const IntervalSelect: React.FC<Props> = ({ defaultSelected, className, onChange }) => {
  const [selected, setSelected] = useState<Interval>(defaultSelected);
  const searchParams = useModifySearchParams();

  return (
    <Select
      value={selected}
      onValueChange={(i: Interval) => {
        onChange?.(i);
        setSelected(i);
        searchParams.set("interval", i);
      }}
    >
      <SelectTrigger className={className}>
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
