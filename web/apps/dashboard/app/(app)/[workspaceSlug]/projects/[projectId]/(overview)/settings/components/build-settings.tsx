"use client";

import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, SettingCard, toast } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

type Props = {
  environmentId: string;
};

const dockerContextSchema = z.object({
  dockerContext: z.string(),
});

const dockerfileSchema = z.object({
  dockerfile: z.string(),
});

const DockerContextCard: React.FC<Props & { defaultValue: string }> = ({
  environmentId,
  defaultValue,
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting },
    control,
  } = useForm<z.infer<typeof dockerContextSchema>>({
    resolver: zodResolver(dockerContextSchema),
    mode: "onChange",
    defaultValues: { dockerContext: defaultValue },
  });

  const currentDockerContext = useWatch({ control, name: "dockerContext" });

  const updateBuild = trpc.deploy.environmentSettings.updateBuild.useMutation({
    onSuccess: () => {
      toast.success("Docker context updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update docker context", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof dockerContextSchema>) => {
    await updateBuild.mutateAsync({
      environmentId,
      dockerContext: values.dockerContext,
    });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Docker Context"
        description="The build context directory for Docker."
        border="top"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <FormInput className="w-full" placeholder="." {...register("dockerContext")} />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateBuild.isLoading ||
              isSubmitting ||
              !isValid ||
              currentDockerContext === defaultValue
            }
            loading={updateBuild.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

const DockerfileCard: React.FC<Props & { defaultValue: string }> = ({
  environmentId,
  defaultValue,
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting },
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
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Dockerfile"
        description="The path to the Dockerfile relative to the context."
        border="bottom"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <FormInput className="w-full" placeholder="Dockerfile" {...register("dockerfile")} />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateBuild.isLoading ||
              isSubmitting ||
              !isValid ||
              currentDockerfile === defaultValue
            }
            loading={updateBuild.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

export const BuildSettings: React.FC<Props> = ({ environmentId }) => {
  const { data } = trpc.deploy.environmentSettings.get.useQuery({ environmentId });

  return (
    <div>
      <DockerContextCard
        environmentId={environmentId}
        defaultValue={data?.buildSettings?.dockerContext ?? ""}
      />
      <DockerfileCard
        environmentId={environmentId}
        defaultValue={data?.buildSettings?.dockerfile ?? ""}
      />
    </div>
  );
};
