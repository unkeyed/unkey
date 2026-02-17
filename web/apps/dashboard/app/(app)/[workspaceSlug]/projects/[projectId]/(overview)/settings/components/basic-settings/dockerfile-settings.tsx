import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileSettings } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";
import { EditableSettingCard } from "../shared/editable-setting-card";

const dockerfileSchema = z.object({
  dockerfile: z.string().min(1, "Dockerfile path is required"),
});

export const DockerfileSettings = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const defaultValue = data?.buildSettings?.dockerfile ?? "Dockerfile";
  return <DockerfileForm environmentId={environmentId} defaultValue={defaultValue} />;
};

const DockerfileForm = ({
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
  } = useForm<z.infer<typeof dockerfileSchema>>({
    resolver: zodResolver(dockerfileSchema),
    mode: "onChange",
    defaultValues: { dockerfile: defaultValue },
  });

  const currentDockerfile = useWatch({ control, name: "dockerfile" });

  const updateBuild = trpc.deploy.environmentSettings.updateBuild.useMutation({
    onSuccess: () => {
      toast.success("Dockerfile updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update dockerfile", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof dockerfileSchema>) => {
    await updateBuild.mutateAsync({ environmentId, dockerfile: values.dockerfile });
  };

  return (
    <EditableSettingCard
      icon={<FileSettings className="text-gray-12" iconSize="xl-medium" />}
      title="Dockerfile"
      description="Dockerfile location used for docker build. (e.g., services/api/Dockerfile)"
      displayValue={defaultValue}
      formId="update-dockerfile-form"
      canSave={isValid && !isSubmitting && currentDockerfile !== defaultValue}
      isSaving={updateBuild.isLoading || isSubmitting}
    >
      <form id="update-dockerfile-form" onSubmit={handleSubmit(onSubmit)}>
        <FormInput
          required
          className="w-[480px]"
          label="Dockerfile path"
          description="Dockerfile location used for docker build"
          placeholder="Dockerfile"
          error={errors.dockerfile?.message}
          variant={errors.dockerfile ? "error" : "default"}
          {...register("dockerfile")}
        />
      </form>
    </EditableSettingCard>
  );
};
