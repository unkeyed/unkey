"use client";
import { Loading } from "@/components/dashboard/loading";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useRouter } from "next/navigation";
import type React from "react";
import { useTransition } from "react";

type Props = {
  selected: string;
  buckets: { name: string }[];
};

export const BucketSelect: React.FC<Props> = ({ buckets, selected }) => {
  const [isPending, startTransition] = useTransition();
  const router = useRouter();

  const options = buckets.map((b) => {
    if (b.name === "unkey_mutations") {
      return { value: b.name, label: "System" };
    }
    return {
      value: b.name,
      label: `Ratelimit: ${b.name}`,
    };
  });

  return (
    <div>
      <Select
        value={selected}
        onValueChange={(value) => {
          startTransition(() => {
            router.push(`/audit/${value}`);
          });
        }}
      >
        {isPending ? (
          <Loading />
        ) : (
          <SelectTrigger disabled={isPending}>
            <SelectValue />
          </SelectTrigger>
        )}
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};
