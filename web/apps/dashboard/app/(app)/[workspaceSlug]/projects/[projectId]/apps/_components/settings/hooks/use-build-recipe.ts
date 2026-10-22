import type { UseFormReturn } from "react-hook-form";
import { z } from "zod";
import { normalizePath } from "../build-settings/path-hint";
import { useSettingForm } from "./use-setting-form";

const dockerContextSegment = /^[A-Za-z0-9._-]+$/;

const buildRecipeSchema = z.object({
  dockerContext: z
    .string()
    .min(1, "Enter a root directory or use '.' for the repository root.")
    .transform((path) => normalizePath(path) || ".")
    .refine(
      (path) =>
        path === "." ||
        (path === path.trim() &&
          !path.includes("\\") &&
          path
            .split("/")
            .every(
              (segment) =>
                segment !== "." && segment !== ".." && dockerContextSegment.test(segment),
            )),
      "Enter a relative path like 'api' or 'services/api'.",
    ),
  dockerfile: z.string().trim(),
  buildCommand: z.string().trim().max(1000),
});

const BUILD_RECIPE_KEYS = ["dockerContext", "dockerfile", "buildCommand"] as const;

export type BuildRecipeForm = UseFormReturn<z.infer<typeof buildRecipeSchema>>;

export function useBuildRecipe() {
  return useSettingForm({
    schema: buildRecipeSchema,
    read: ({ dockerContext, dockerfile, buildCommand }) => ({
      dockerContext,
      dockerfile,
      buildCommand,
    }),
    write: (draft, values, saved) => {
      for (const key of BUILD_RECIPE_KEYS) {
        if (values[key] !== saved[key]) {
          draft[key] = values[key];
        }
      }
    },
  });
}
