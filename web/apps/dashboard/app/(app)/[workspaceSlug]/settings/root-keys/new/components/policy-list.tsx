"use client";

import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRef, useState } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { type Policy, isPolicyComplete } from "../lib/policy";
import type { RootKeyFormValues } from "../schema";
import { PolicyCard } from "./policy-card";
import { TemplateGallery } from "./template-gallery";

type PolicyListProps = {
  showErrors: boolean;
};

export function PolicyList({ showErrors }: PolicyListProps) {
  const { control, setValue } = useFormContext<RootKeyFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "policies" });
  const policies = useWatch({ control, name: "policies" }) ?? [];
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [adding, setAdding] = useState(false);
  const showGallery = adding || fields.length === 0;

  const seeded = useRef<Record<string, boolean>>({});

  const isCollapsed = (id: string, policy: Policy) => {
    seeded.current[id] ??= isPolicyComplete(policy);
    return collapsed[id] ?? seeded.current[id];
  };

  const openGallery = () => {
    setCollapsed((current) =>
      fields.reduce((next, field) => ({ ...next, [field.id]: true }), current),
    );
    setAdding(true);
  };

  const pick = (picked: Policy[]) => {
    append(picked, { shouldFocus: false });
    setAdding(false);
  };

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
            collapsed={isCollapsed(field.id, policy)}
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
        <span className="text-[13px] leading-5 text-error-11">Grant at least one permission.</span>
      ) : null}

      {showGallery ? (
        <TemplateGallery
          onPick={pick}
          onCancel={fields.length > 0 ? () => setAdding(false) : undefined}
        />
      ) : (
        <div>
          <Button
            type="button"
            variant="ghost"
            size="md"
            className="font-medium"
            onClick={openGallery}
          >
            <Plus />
            Add policy
          </Button>
        </div>
      )}
    </div>
  );
}
