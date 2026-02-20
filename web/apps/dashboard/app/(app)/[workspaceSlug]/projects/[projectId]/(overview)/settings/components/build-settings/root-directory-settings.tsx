import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { FolderLink } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const rootDirectorySchema = z.object({
  dockerContext: z.string(),
});

export const RootDirectorySettings = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data: settings } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, environmentId ?? "")),
    [environmentId],
  );

  const defaultValue = settings?.[0]?.dockerContext ?? ".";
  return <RootDirectoryForm environmentId={environmentId} defaultValue={defaultValue} />;
};

const RootDirectoryForm = ({
  environmentId,
  defaultValue,
}: {
  environmentId: string;
  defaultValue: string;
}) => {
  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
  } = useForm<z.infer<typeof rootDirectorySchema>>({
    resolver: zodResolver(rootDirectorySchema),
    mode: "onChange",
    defaultValues: { dockerContext: defaultValue },
  });

  const currentDockerContext = useWatch({ control, name: "dockerContext" });

  const onSubmit = async (values: z.infer<typeof rootDirectorySchema>) => {
    if (!environmentId) {
      return;
    }
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.dockerContext = values.dockerContext;
    });
  };

  return (
    <FormSettingCard
      icon={<FolderLink className="text-gray-12" iconSize="xl-medium" />}
      title="Root directory"
      description="Build context directory. All COPY/ADD commands are relative to this path. (e.g., services/api)"
      displayValue={defaultValue || "."}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && currentDockerContext !== defaultValue}
      isSaving={isSubmitting}
    >
      <FormInput
        label="Root directory"
        required
        className="w-[480px]"
        description="Build context directory for Docker. Changes apply on next deploy."
        placeholder="."
        error={errors.dockerContext?.message}
        variant={errors.dockerContext ? "error" : "default"}
        {...register("dockerContext")}
      />
    </FormSettingCard>
  );
};
