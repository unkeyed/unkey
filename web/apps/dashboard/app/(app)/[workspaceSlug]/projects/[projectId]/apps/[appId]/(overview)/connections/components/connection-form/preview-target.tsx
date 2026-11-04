"use client";

import { IconPlusOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useFormContext } from "react-hook-form";
import type { CallerSnapshot } from "../../hooks/use-connections-data";
import type { ConnectionFormValues } from "../../model/form";
import type { TargetRule } from "../../model/target";
import { TargetField } from "./target-field";

export function PreviewTarget({
  caller,
  target,
  rule,
}: {
  caller: CallerSnapshot;
  target: { id: string; name: string };
  rule: TargetRule | null;
}) {
  const { getValues, setValue, trigger } = useFormContext<ConnectionFormValues>();
  const connect = (next: TargetRule | null) => {
    setValue("targets.preview", next, { shouldDirty: true, shouldValidate: false });
    if (getValues("name").trim() !== "") {
      void trigger("name");
    }
  };

  if (!rule) {
    return (
      <Button
        type="button"
        variant="outline"
        size="md"
        className="w-full"
        onClick={() => connect({ targetType: "automatic" })}
      >
        <IconPlusOutline18 />
        Connect preview deployments
      </Button>
    );
  }
  return (
    <TargetField
      caller={caller}
      target={target}
      rule={rule}
      labelAction={
        <Button type="button" variant="ghost" size="sm" onClick={() => connect(null)}>
          Remove
        </Button>
      }
    />
  );
}
