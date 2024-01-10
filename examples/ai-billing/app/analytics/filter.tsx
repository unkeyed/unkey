"use client";

import { DateRangePicker } from "@tremor/react";
import { useRouter } from "next/navigation";
import React from "react";

export function FilterDateRange({
  defaultStart,
  defaultEnd,
}: { defaultStart?: number; defaultEnd?: number }) {
  const router = useRouter();
  return (
    <DateRangePicker
      enableClear
      enableSelect
      defaultValue={{
        from: defaultStart ? new Date(defaultStart) : undefined,
        to: defaultEnd ? new Date(defaultEnd) : undefined,
      }}
      maxDate={new Date()}
      onValueChange={(v) => {
        const params = new URLSearchParams();
        if (v.from) {
          params.set("start", v.from.getTime().toString());
        }
        if (v.to) {
          params.set("end", v.to.getTime().toString());
        }
        router.push(`/analytics?${params.toString()}`);
      }}
    />
  );
}
