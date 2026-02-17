import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { FolderLink } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../data-provider";
import { EditableSettingCard } from "./editable-setting-card";

const rootDirectorySchema = z.object({
  dockerContext: z.string(),
});

export const RootDirectorySettings = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const defaultValue = data?.buildSettings?.dockerContext ?? ".";
  return <RootDirectoryForm environmentId={environmentId} defaultValue={defaultValue} />;
};

const RootDirectoryForm = ({
  environmentId,
  defaultValue,
}: {
  environmentId: string;
  defaultValue: string;
}) => {
  const utils = trpc.useUtils();

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

  const updateBuild = trpc.deploy.environmentSettings.updateBuild.useMutation({
    onSuccess: () => {
      toast.success("Root directory updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update root directory", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof rootDirectorySchema>) => {
    await updateBuild.mutateAsync({ environmentId, dockerContext: values.dockerContext });
  };

  return (
    <EditableSettingCard
      icon={<FolderLink className="text-gray-12" iconSize="xl-medium" />}
      title="Root directory"
      description="Build context directory. All COPY/ADD commands are relative to this path. (e.g., services/api)"
      border="bottom"
      displayValue={currentDockerContext || "."}
      formId="update-root-directory-form"
      canSave={isValid && !isSubmitting && currentDockerContext !== defaultValue}
      isSaving={updateBuild.isLoading || isSubmitting}
    >
      <form id="update-root-directory-form" onSubmit={handleSubmit(onSubmit)}>
        <FormInput
          label="Root directory"
          description="Build context directory for Docker"
          placeholder="."
          error={errors.dockerContext?.message}
          variant={errors.dockerContext ? "error" : "default"}
          {...register("dockerContext")}
        />
      </form>
    </EditableSettingCard>
  );
};
