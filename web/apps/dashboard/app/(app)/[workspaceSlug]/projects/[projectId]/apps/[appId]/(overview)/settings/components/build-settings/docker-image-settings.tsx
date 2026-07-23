"use client";

import { collection } from "@/lib/collections";
import type { App } from "@/lib/collections/deploy/apps";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Cube } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const schema = z.object({
  image: z.string().trim().min(1, "Image is required").max(512, "Image is too long"),
});

export const DockerImage = ({ app }: { app: App }) => {
  const defaultValue = app.defaultDockerImage ?? "";
  const update = trpc.deploy.app.updateDockerImage.useMutation();
  const {
    register,
    handleSubmit,
    control,
    formState: { errors, isSubmitting, isValid },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    mode: "onChange",
    defaultValues: { image: defaultValue },
  });
  const image = useWatch({ control, name: "image" });
  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [image.trim() === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof schema>) => {
    await update.mutateAsync({ appId: app.id, image: values.image });
    await collection.apps.utils.refetch();
    toast.success("Docker image updated", {
      description: "The new image will be used for future deployments.",
    });
  };

  return (
    <FormSettingCard
      icon={<Cube className="text-gray-12" iconSize="xl-medium" />}
      title="Docker image"
      description="The default container image used for new deployments."
      displayValue={defaultValue || "Not configured"}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <SettingField>
        <FormInput
          label="Image"
          requirement="required"
          description="Use a full image reference with a tag or digest."
          placeholder="ghcr.io/acme/my-app:latest"
          error={errors.image?.message}
          {...register("image")}
        />
      </SettingField>
    </FormSettingCard>
  );
};
