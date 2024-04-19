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
  ratelimitNamespaces: { id: string; name: string }[];
};

export const BucketSelect: React.FC<Props> = ({ ratelimitNamespaces, selected }) => {
  const [isPending, startTransition] = useTransition();
  const router = useRouter();

  const options = [
    { value: "unkey_mutations", label: "System" },
    ...ratelimitNamespaces.map((ns) => ({
      value: ns.id,
      label: `Ratelimit: ${ns.name}`,
    })),
  ];

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
