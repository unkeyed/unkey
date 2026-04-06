"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { XMark } from "@unkey/icons";
import type { Control } from "react-hook-form";
import { useController } from "react-hook-form";
import type { PolicyFormValues } from "../schema";

type KeyauthFormValues = Extract<PolicyFormValues, { type: "keyauth" }>;

export function KeyAuthFields({ control }: { control: Control<KeyauthFormValues> }) {
  const {
    field: { value: keySpaceIds, onChange: setKeySpaceIds },
    fieldState: { error },
  } = useController({ control, name: "keySpaceIds" });

  const { data: availableKeyspaces = {} } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();

  const unselected = Object.keys(availableKeyspaces).filter((id) => !keySpaceIds.includes(id));
  const comboboxOptions: ComboboxOption[] = unselected.map((id) => ({
    value: id,
    searchValue: id,
    label: (
      <span className="text-gray-11 text-xs font-mono">
        {availableKeyspaces[id]?.api?.name ?? id}
      </span>
    ),
  }));

  return (
    <div className="flex flex-col gap-1.5">
      <FormCombobox
        label="Keyspaces"
        description="API keyspaces used to authenticate incoming requests."
        options={comboboxOptions}
        value=""
        onSelect={(id) => {
          if (!keySpaceIds.includes(id)) {
            setKeySpaceIds([...keySpaceIds, id]);
          }
        }}
        placeholder={
          keySpaceIds.length === 0 ? (
            <span className="text-grayA-8 w-full text-left">Select a keyspace</span>
          ) : (
            <div className="w-full flex flex-wrap gap-1.5 py-0.5">
              {keySpaceIds.map((id) => (
                <span
                  key={id}
                  className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-grayA-3 border border-grayA-4 text-xs text-accent-12"
                >
                  {availableKeyspaces[id]?.api?.name ?? id}
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      setKeySpaceIds(keySpaceIds.filter((k) => k !== id));
                    }}
                    className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors"
                  >
                    <XMark iconSize="sm-regular" />
                  </button>
                </span>
              ))}
            </div>
          )
        }
        searchPlaceholder="Search keyspaces..."
        emptyMessage={<div className="mt-2">No keyspaces available.</div>}
      />
      {error && <p className="text-error-11 text-[13px]">{error.message}</p>}
    </div>
  );
}
