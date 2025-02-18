"use client";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { parseAsString, useQueryState } from "nuqs";
import type React from "react";

type Props = {
  buckets: { id: string; name: string }[];
};

export const BucketSelect: React.FC<Props> = ({ buckets }) => {
  const [selected, setSelected] = useQueryState(
    "bucket",
    parseAsString.withDefault("unkey_mutations").withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );

  return (
    <div>
      <Select
        value={selected}
        onValueChange={(value) => {
          setSelected(value);
        }}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {buckets.map((b) => (
            <SelectItem key={b.id} value={b.name}>
              {b.name === "unkey_mutations" ? "System" : `Ratelimit: ${b.name}`}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};
