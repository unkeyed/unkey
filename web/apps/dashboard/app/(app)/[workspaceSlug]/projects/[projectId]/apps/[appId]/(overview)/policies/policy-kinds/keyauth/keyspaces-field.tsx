"use client";

import { FormCombobox } from "@/components/ui/form-combobox";
import { IconXmarkOutline12 } from "@unkey/icons";
import type { ComboboxOption } from "@unkey/ui";
import { useController, useFormContext } from "react-hook-form";
import type { KeyauthFormValues } from "./model";
import { useKeyspaceNames } from "./use-keyspace-names";

export function KeyspacesField() {
  const { control } = useFormContext<KeyauthFormValues>();
  const {
    field: { value: keyspaceIds, onChange: setKeyspaceIds },
    fieldState: { error },
  } = useController({ control, name: "keyspaceIds" });
  const names = useKeyspaceNames();
  const nameOf = (id: string) => names[id] ?? id;

  const remove = (id: string) => setKeyspaceIds(keyspaceIds.filter((k) => k !== id));

  const options: ComboboxOption[] = Object.keys(names)
    .filter((id) => !keyspaceIds.includes(id))
    .map((id) => ({
      value: id,
      searchValue: id,
      label: <span className="text-gray-11 text-xs font-mono">{nameOf(id)}</span>,
    }));

  return (
    <div className="flex flex-col gap-1.5">
      <FormCombobox
        label="Keyspaces"
        error={error?.message}
        options={options}
        value=""
        onSelect={(id) => {
          if (!keyspaceIds.includes(id)) {
            setKeyspaceIds([...keyspaceIds, id]);
          }
        }}
        placeholder={
          keyspaceIds.length === 0 ? (
            <span className="text-grayA-8 w-full text-left">Select a keyspace</span>
          ) : (
            <div className="w-full flex flex-wrap gap-1.5 py-0.5">
              {keyspaceIds.map((id) => (
                <span
                  key={id}
                  className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-grayA-3 border text-xs text-gray-12"
                >
                  {nameOf(id)}
                  {/* biome-ignore lint/a11y/useSemanticElements: nested inside a <button> (combobox trigger), so <button> is invalid here */}
                  <span
                    role="button"
                    tabIndex={0}
                    aria-label={`Remove ${nameOf(id)}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      remove(id);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.stopPropagation();
                        remove(id);
                      }
                    }}
                    className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-gray-12 transition-colors cursor-pointer"
                  >
                    <IconXmarkOutline12 />
                  </span>
                </span>
              ))}
            </div>
          )
        }
        searchPlaceholder="Search keyspaces..."
        emptyMessage={<div className="mt-2">No keyspaces available.</div>}
      />
    </div>
  );
}
