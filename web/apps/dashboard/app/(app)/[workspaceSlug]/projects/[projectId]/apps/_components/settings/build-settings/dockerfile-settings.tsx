import { FormCombobox } from "@/components/ui/form-combobox";
import { SettingsForm, SettingsRow } from "@unkey/ui";
import { useMemo } from "react";
import { z } from "zod";
import { useSettingForm } from "../hooks/use-setting-form";
import { pathHint, pathInputVariant } from "./path-hint";
import { pathHintMessage } from "./path-hint-message";
import { useRepoTree } from "./use-repo-tree";

const dockerfileSchema = z.object({
  // Empty means "no Dockerfile configured": the app builds with Railpack.
  dockerfile: z.string(),
});

export function Dockerfile() {
  const { settings, form, formProps } = useSettingForm({
    schema: dockerfileSchema,
    read: (s) => ({ dockerfile: s.dockerfile }),
    write: (draft, values) => {
      draft.dockerfile = values.dockerfile;
    },
  });
  const { dockerContext } = settings;
  const { branch, validateDockerfilePath, findDockerfileCaseMatch, getDockerfilesForContext } =
    useRepoTree();

  const currentDockerfile = form.watch("dockerfile");
  const setDockerfile = (path: string) =>
    form.setValue("dockerfile", path, { shouldValidate: true });

  // An empty path is valid: it means the app builds with Railpack instead.
  const validation = currentDockerfile
    ? validateDockerfilePath(currentDockerfile, dockerContext)
    : "valid";
  const hint = pathHint(
    validation,
    () => findDockerfileCaseMatch(currentDockerfile, dockerContext),
    branch,
  );
  const detectedDockerfiles = getDockerfilesForContext(dockerContext);

  const options = useMemo(
    () => [
      // The empty value is a first-class choice: it means "no Dockerfile
      // configured" and the app is built with Railpack instead. searchValue
      // keeps the option matchable since cmdk can't match an empty string.
      {
        label: <span className="text-gray-11">Automatic (no Dockerfile)</span>,
        selectedLabel: "Automatic (no Dockerfile)",
        value: "",
        searchValue: "automatic (no dockerfile)",
      },
      ...detectedDockerfiles.map((path) => ({
        label: path,
        selectedLabel: path,
        value: path,
        searchValue: path,
      })),
    ],
    [detectedDockerfiles],
  );

  return (
    <SettingsForm {...formProps}>
      <SettingsRow title="Dockerfile">
        <FormCombobox
          aria-label="Dockerfile"
          description={pathHintMessage(hint, "File", setDockerfile)}
          options={options}
          value={currentDockerfile}
          onSelect={setDockerfile}
          creatable
          searchPlaceholder="Search or type a path..."
          emptyMessage={<div className="mt-2">No Dockerfiles detected in repository</div>}
          placeholder={<span className="text-grayA-8">None (auto-detected build)</span>}
          variant={pathInputVariant(Boolean(form.formState.errors.dockerfile), hint)}
        />
      </SettingsRow>
    </SettingsForm>
  );
}
