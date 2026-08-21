"use client";

import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { newPolicy } from "../lib/policy";
import type { RootKeyFormValues } from "../schema";
import { PolicyCard } from "./policy-card";

type PolicyListProps = {
  showErrors: boolean;
};

export function PolicyList({ showErrors }: PolicyListProps) {
  const { control, setValue } = useFormContext<RootKeyFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "policies" });
  const policies = useWatch({ control, name: "policies" }) ?? [];
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  return (
    <div className="flex flex-col gap-3">
      {fields.map((field, index) => {
        const policy = policies[index];
        if (!policy) {
          return null;
        }
        return (
          <PolicyCard
            key={field.id}
            policy={policy}
            collapsed={collapsed[field.id] === true}
            showError={showErrors}
            onChange={(next) =>
              setValue(`policies.${index}`, next, { shouldDirty: true, shouldValidate: true })
            }
            onRemove={() => remove(index)}
            onCollapsedChange={(next) =>
              setCollapsed((current) => ({ ...current, [field.id]: next }))
            }
          />
        );
      })}

      {showErrors && fields.length === 0 ? (
        <span className="text-xs text-error-11">Grant at least one permission.</span>
      ) : null}

      <div>
        <Button
          type="button"
          variant="ghost"
          size="md"
          className="font-medium"
          onClick={() => append(newPolicy())}
        >
          <Plus iconSize="sm-regular" />
          Add policy
        </Button>
      </div>
    </div>
  );
}
