"use client";

import { DateRangePicker, TextInput } from "@tremor/react";
import { useRouter } from "next/navigation";
import React from "react";

export const FilterByOwnerId: React.FC<{ defaultOwnerId?: string }> = ({ defaultOwnerId }) => {
  const router = useRouter();
  return (
    <TextInput
      defaultValue={defaultOwnerId}
      onChange={(value) => {
        router.push(`/analytics?ownerId=${value.currentTarget.value}`);
      }}
      className="w-full"
      placeholder="Filter by Owner"
    />
  );
};
export const FilterByKeyId: React.FC<{ defaultKeyId?: string }> = ({ defaultKeyId }) => {
  const router = useRouter();
  return (
    <TextInput
      defaultValue={defaultKeyId}
      onChange={(value) => {
        router.push(`/analytics?keyId=${value.currentTarget.value}`);
      }}
      className="w-full"
      placeholder="Filter by Key Id"
    />
  );
};

export const FilterDateRange: React.FC<{
  defaultStart?: number;
  defaultEnd?: number;
}> = ({ defaultStart, defaultEnd }) => {
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
};
