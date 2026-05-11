"use client";

import { Switch } from "@/components/ui/switch";
import type {
  ExtensionConfigField,
  ExtensionConfigState,
  ExtensionConfigValue,
} from "@/lib/extensions/registry";
import { Plus, Trash } from "@unkey/icons";
import { Button, Checkbox, FormInput } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

type ConfigFormProps = {
  fields: ExtensionConfigField[];
  values: ExtensionConfigState;
  onChange: (next: ExtensionConfigState) => void;
};

export function ConfigForm({ fields, values, onChange }: ConfigFormProps) {
  const visibleFields = fields.filter((field) => isFieldVisible(field, values));
  const groups = groupFields(visibleFields);

  if (fields.length === 0) {
    return (
      <p className="text-[12px] text-grayA-10">This extension has no configuration options.</p>
    );
  }

  const setField = (id: string, value: ExtensionConfigValue) => {
    onChange({ ...values, [id]: value });
  };

  return (
    <div className="flex flex-col gap-5">
      {groups.map((group) => (
        <fieldset key={group.label ?? "_"} className="flex flex-col gap-3">
          {group.label ? (
            <legend className="text-[11px] font-medium uppercase tracking-wide text-grayA-10">
              {group.label}
            </legend>
          ) : null}
          <div className="flex flex-col gap-4">
            {group.fields.map((field) => (
              <Field
                key={field.id}
                field={field}
                value={values[field.id]}
                onChange={(v) => setField(field.id, v)}
              />
            ))}
          </div>
        </fieldset>
      ))}
    </div>
  );
}

function Field({
  field,
  value,
  onChange,
}: {
  field: ExtensionConfigField;
  value: ExtensionConfigValue | undefined;
  onChange: (value: ExtensionConfigValue) => void;
}) {
  switch (field.type) {
    case "text":
    case "url":
    case "secret":
      return (
        <FormInput
          label={fieldLabel(field)}
          type={field.type === "secret" ? "password" : field.type === "url" ? "url" : "text"}
          placeholder={field.placeholder}
          description={field.helpText}
          value={typeof value === "string" ? value : ""}
          onChange={(e) => onChange(e.target.value)}
        />
      );
    case "select": {
      const selectId = `extension-field-${field.id}`;
      return (
        <div className="flex flex-col gap-1.5">
          <label htmlFor={selectId} className="text-[12px] font-medium text-grayA-12">
            {fieldLabel(field)}
          </label>
          <select
            id={selectId}
            value={typeof value === "string" ? value : ""}
            onChange={(e) => onChange(e.target.value)}
            className="h-9 rounded-lg border border-grayA-3 bg-background px-2 text-[13px] text-grayA-12 hover:border-grayA-6"
          >
            {field.options?.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          {field.helpText ? (
            <span className="text-[11px] text-grayA-10">{field.helpText}</span>
          ) : null}
        </div>
      );
    }
    case "boolean":
      return (
        <div className="flex items-start justify-between gap-4 rounded-md border border-grayA-3 p-3">
          <div className="flex flex-col gap-0.5">
            <span className="text-[13px] font-medium text-grayA-12">{fieldLabel(field)}</span>
            {field.helpText ? (
              <span className="text-[12px] text-grayA-10">{field.helpText}</span>
            ) : null}
          </div>
          <Switch
            checked={value === true}
            onCheckedChange={(checked) => onChange(checked === true)}
          />
        </div>
      );
    case "multiselect": {
      const selected = Array.isArray(value) ? value : [];
      return (
        <div className="flex flex-col gap-2">
          <span className="text-[12px] font-medium text-grayA-12">{fieldLabel(field)}</span>
          <div className="flex flex-col gap-2">
            {field.options?.map((option) => {
              const isOn = selected.includes(option.value);
              const toggle = () => {
                const next = isOn
                  ? selected.filter((v) => v !== option.value)
                  : [...selected, option.value];
                onChange(next);
              };
              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={toggle}
                  aria-pressed={isOn}
                  className={cn(
                    "flex items-center gap-3 rounded-md border p-3 text-left cursor-pointer transition-colors",
                    isOn ? "border-grayA-6 bg-grayA-2" : "border-grayA-3 hover:border-grayA-5",
                  )}
                >
                  <Checkbox
                    checked={isOn}
                    onCheckedChange={() => toggle()}
                    onClick={(e) => e.stopPropagation()}
                  />
                  <span className="flex flex-col gap-0.5">
                    <span className="text-[13px] font-medium text-grayA-12">{option.label}</span>
                    {option.description ? (
                      <span className="text-[12px] text-grayA-10">{option.description}</span>
                    ) : null}
                  </span>
                </button>
              );
            })}
          </div>
          {field.helpText ? (
            <span className="text-[11px] text-grayA-10">{field.helpText}</span>
          ) : null}
        </div>
      );
    }
    case "string-list": {
      const items = Array.isArray(value) ? value : [];
      return (
        <StringListField
          label={fieldLabel(field)}
          placeholder={field.placeholder}
          helpText={field.helpText}
          values={items}
          onChange={onChange}
        />
      );
    }
  }
}

function StringListField({
  label,
  placeholder,
  helpText,
  values,
  onChange,
}: {
  label: string;
  placeholder?: string;
  helpText?: string;
  values: string[];
  onChange: (next: string[]) => void;
}) {
  const update = (index: number, next: string) => {
    onChange(values.map((v, i) => (i === index ? next : v)));
  };
  const remove = (index: number) => {
    onChange(values.filter((_, i) => i !== index));
  };
  const add = () => {
    onChange([...values, ""]);
  };

  return (
    <div className="flex flex-col gap-2">
      <span className="text-[12px] font-medium text-grayA-12">{label}</span>
      <div className="flex flex-col gap-2">
        {values.length === 0 ? (
          <p className="text-[12px] text-grayA-10">No entries.</p>
        ) : (
          values.map((entry, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: free-form list, no stable id available
            <div key={index} className="flex items-center gap-2">
              <FormInput
                placeholder={placeholder}
                value={entry}
                onChange={(e) => update(index, e.target.value)}
                wrapperClassName="flex-1"
              />
              <Button
                variant="ghost"
                color="danger"
                onClick={() => remove(index)}
                className="size-9"
              >
                <Trash className="size-4" />
              </Button>
            </div>
          ))
        )}
      </div>
      <Button variant="outline" size="sm" onClick={add} className="w-fit">
        <Plus className="size-4" />
        Add entry
      </Button>
      {helpText ? <span className="text-[11px] text-grayA-10">{helpText}</span> : null}
    </div>
  );
}

function fieldLabel(field: ExtensionConfigField): string {
  return field.required ? `${field.label} *` : field.label;
}

function groupFields(
  fields: ExtensionConfigField[],
): Array<{ label: string | undefined; fields: ExtensionConfigField[] }> {
  const order: Array<string | undefined> = [];
  const buckets = new Map<string | undefined, ExtensionConfigField[]>();
  for (const field of fields) {
    const key = field.group;
    if (!buckets.has(key)) {
      buckets.set(key, []);
      order.push(key);
    }
    buckets.get(key)?.push(field);
  }
  return order.map((label) => ({ label, fields: buckets.get(label) ?? [] }));
}

function isFieldVisible(field: ExtensionConfigField, values: ExtensionConfigState): boolean {
  if (!field.dependsOn) {
    return true;
  }
  const target = values[field.dependsOn.fieldId];
  const expected = field.dependsOn.equals;
  if (Array.isArray(target)) {
    if (typeof expected === "string") {
      return target.includes(expected);
    }
    return false;
  }
  return target === expected;
}

/**
 * Build the initial config state for an extension by applying any
 * declared `defaultValue`s. Used by the install wizard so the form
 * starts in a sensible state.
 */
export function initialConfigState(fields: ExtensionConfigField[]): ExtensionConfigState {
  const state: ExtensionConfigState = {};
  for (const field of fields) {
    if (field.defaultValue !== undefined) {
      state[field.id] = field.defaultValue;
    } else if (field.type === "multiselect" || field.type === "string-list") {
      state[field.id] = [];
    } else if (field.type === "boolean") {
      state[field.id] = false;
    } else {
      state[field.id] = "";
    }
  }
  return state;
}

/**
 * Returns ids of required fields that are missing/empty, taking
 * conditional visibility into account.
 */
export function validateConfigState(
  fields: ExtensionConfigField[],
  values: ExtensionConfigState,
): string[] {
  const errors: string[] = [];
  for (const field of fields) {
    if (!field.required) {
      continue;
    }
    if (!isFieldVisible(field, values)) {
      continue;
    }
    const value = values[field.id];
    if (value === undefined) {
      errors.push(field.id);
      continue;
    }
    if (typeof value === "string" && value.trim().length === 0) {
      errors.push(field.id);
    } else if (Array.isArray(value) && value.length === 0) {
      errors.push(field.id);
    }
  }
  return errors;
}
