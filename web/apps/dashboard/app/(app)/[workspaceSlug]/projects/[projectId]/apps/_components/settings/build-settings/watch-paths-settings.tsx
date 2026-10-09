"use client";

import { FormCombobox } from "@/components/ui/form-combobox";
import { sameJson } from "@/lib/utils/same-json";
import { IconPlusOutline18 } from "@unkey/icons";
import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { useCallback, useRef } from "react";
import { useFieldArray } from "react-hook-form";
import { z } from "zod";
import { RemoveButton } from "../../remove-button";
import { useSettingForm } from "../hooks/use-setting-form";
import { useRepoTree } from "./use-repo-tree";

const watchPathsSchema = z.object({
  paths: z.array(
    z.object({
      value: z
        .string()
        .refine(
          (value) => value === "" || isValidWatchPathPattern(value),
          "Not a valid glob pattern. Use syntax like 'src/**' or '**/*.go'.",
        ),
    }),
  ),
});

type WatchPathsForm = z.infer<typeof watchPathsSchema>;

function fromFormPaths(paths: { value: string }[]): string[] {
  return paths.map((p) => p.value).filter(Boolean);
}

export function WatchPaths() {
  const { settings, form, formProps } = useSettingForm({
    schema: watchPathsSchema,
    read: (s) => ({ paths: s.watchPaths.map((value) => ({ value })) }),
    write: (draft, values) => {
      draft.watchPaths = fromFormPaths(values.paths);
    },
    isEqual: (current: WatchPathsForm, saved: WatchPathsForm) =>
      sameJson(fromFormPaths(current.paths), fromFormPaths(saved.paths)),
  });
  const { watchPathSuggestions } = useRepoTree();
  const { control, register, setValue, trigger } = form;
  const { errors } = form.formState;

  const { fields, append, remove } = useFieldArray({
    control,
    name: "paths",
  });

  const inputRefs = useRef<Map<number, HTMLInputElement>>(new Map());
  const setInputRef = useCallback((index: number, el: HTMLInputElement | null) => {
    if (el) {
      inputRefs.current.set(index, el);
    } else {
      inputRefs.current.delete(index);
    }
  }, []);

  const removeAndFocus = useCallback(
    (index: number) => {
      remove(index);
      const focusIndex = index > 0 ? index - 1 : 0;
      requestAnimationFrame(() => {
        inputRefs.current.get(focusIndex)?.focus();
      });
    },
    [remove],
  );

  const appendAndFocus = useCallback(() => {
    append({ value: "" });
    inputRefs.current.get(fields.length)?.focus();
  }, [append, fields.length]);

  const currentPaths = form.watch("paths");
  const currentValues = fromFormPaths(currentPaths);
  const rootWatchPath =
    settings.dockerContext && settings.dockerContext !== "."
      ? `${settings.dockerContext}/**`
      : null;
  const options = [...watchPathSuggestions]
    .sort((a, b) => {
      if (a.path === rootWatchPath) {
        return -1;
      }
      if (b.path === rootWatchPath) {
        return 1;
      }
      return a.path.localeCompare(b.path);
    })
    .map(({ path, marker }) => ({
      label: (
        <span className="flex w-full items-center justify-between gap-4">
          <span className="truncate font-mono">{path}</span>
          <span className="shrink-0 text-gray-9">
            {path === rootWatchPath ? "Root directory" : marker}
          </span>
        </span>
      ),
      selectedLabel: path,
      value: path,
      searchValue: path,
      disabled: currentValues.includes(path),
    }));

  const addWatchPath = (value: string) => {
    if (!value || currentValues.includes(value)) {
      return;
    }

    const emptyIndex = currentPaths.findIndex((path) => !path.value);
    if (emptyIndex >= 0) {
      setValue(`paths.${emptyIndex}.value`, value, { shouldValidate: true });
      return;
    }
    // useFieldArray's append() doesn't run the resolver, unlike setValue's
    // shouldValidate above, so the newly appended field needs an explicit trigger.
    const newIndex = fields.length;
    append({ value });
    trigger(`paths.${newIndex}.value`);
  };

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Watch paths</SettingsRowTitle>
          <SettingsRowDescription>Only deploy when these files change.</SettingsRowDescription>
        </SettingsRowHeader>
        <SettingsRowContent className="flex max-w-(--setting-w) flex-col gap-1.5">
          {fields.map((field, index) => {
            const { ref: rhfRef, ...fieldProps } = register(`paths.${index}.value`);
            return (
              <div key={field.id} className="flex items-start gap-2">
                <FormInput
                  data-1p-ignore
                  className="flex-1 [&_input]:font-mono"
                  placeholder="src/**"
                  error={errors.paths?.[index]?.value?.message}
                  {...fieldProps}
                  ref={(el: HTMLInputElement | null) => {
                    rhfRef(el);
                    setInputRef(index, el);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      appendAndFocus();
                    }
                    if (e.key === "Backspace" && !currentPaths[index]?.value) {
                      e.preventDefault();
                      removeAndFocus(index);
                    }
                  }}
                />
                <RemoveButton
                  label="Remove watch path"
                  onClick={() => removeAndFocus(index)}
                  className="shrink-0 transition-opacity duration-150"
                />
              </div>
            );
          })}
          <FormCombobox
            options={options}
            value=""
            onSelect={addWatchPath}
            creatable
            leftIcon={<IconPlusOutline18 />}
            searchPlaceholder="Search suggestions or enter a glob..."
            emptyMessage={<div className="mt-2">No suggested watch paths detected</div>}
            placeholder={
              <span className="text-grayA-8">
                {fields.length === 0 ? "All files" : "Add a watch path..."}
              </span>
            }
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

// Mirrors doublestar.ValidatePattern (github.com/bmatcuk/doublestar/v4,
// validate.go): a pure syntax check for balanced [ ] / { } and a non-trailing
// backslash escape.
function isValidWatchPathPattern(pattern: string): boolean {
  let altDepth = 0;
  const len = pattern.length;

  for (let i = 0; i < len; i++) {
    const ch = pattern[i];

    if (ch === "\\") {
      i++;
      if (i >= len) {
        return false;
      }
      continue;
    }

    if (ch === "[") {
      i++;
      if (i >= len) {
        return false;
      }
      if (pattern[i] === "^" || pattern[i] === "!") {
        i++;
      }
      if (i >= len || pattern[i] === "]") {
        return false;
      }

      let closed = false;
      for (; i < len; i++) {
        if (pattern[i] === "\\") {
          i++;
        } else if (pattern[i] === "]") {
          closed = true;
          break;
        }
      }
      if (!closed) {
        return false;
      }
      continue;
    }

    if (ch === "{") {
      altDepth++;
      continue;
    }

    if (ch === "}") {
      if (altDepth === 0) {
        return false;
      }
      altDepth--;
    }
  }

  return altDepth === 0;
}
