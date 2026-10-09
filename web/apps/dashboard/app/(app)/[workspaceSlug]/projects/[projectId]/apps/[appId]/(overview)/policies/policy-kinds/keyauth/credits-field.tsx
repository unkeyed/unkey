"use client";

import { FormDescription, FormInput } from "@unkey/ui";
import { useController, useFormContext } from "react-hook-form";
import { FieldSection, RemoveRowButton, parseOptionalInt } from "../field-parts";
import type { KeyauthFormValues } from "./model";

export function CreditCostField() {
  const { control } = useFormContext<KeyauthFormValues>();
  const {
    field: { value: credits, onChange: setCredits },
    fieldState: { error },
  } = useController({ control, name: "credits" });

  return (
    <FieldSection
      label="Credit Cost"
      htmlFor="keyauth-credits"
      tooltip="Credits each matching request deducts from the key. Keys with unlimited usage are unaffected."
      onAdd={credits === undefined ? () => setCredits(0) : null}
    >
      {credits !== undefined ? (
        <>
          <div className="flex items-center gap-2">
            <FormInput
              id="keyauth-credits"
              type="number"
              min={0}
              placeholder="0"
              // Once added, the override is always a number; an empty input
              // maps to 0 rather than collapsing the section.
              value={credits}
              onChange={(e) => setCredits(parseOptionalInt(e.target.value) ?? 0)}
              className="flex-1"
              variant={error ? "error" : undefined}
              aria-invalid={Boolean(error)}
            />
            <RemoveRowButton label="Remove credit cost" onClick={() => setCredits(undefined)} />
          </div>
          <FormDescription
            error={error?.message}
            descriptionId="keyauth-credits-desc"
            errorId="keyauth-credits-error"
            description="Set to 0 to verify the key without spending credits."
          />
        </>
      ) : null}
    </FieldSection>
  );
}
