"use client";

import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingCard,
  toast,
} from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

type Props = {
  environmentId: string;
};

const portSchema = z.object({
  port: z.number().min(2000).max(54000),
});

const commandSchema = z.object({
  command: z.string(),
});

const healthcheckSchema = z.object({
  method: z.enum(["GET", "POST"]),
  path: z.string(),
});

const PortCard: React.FC<Props & { defaultPort: number }> = ({ environmentId, defaultPort }) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { errors, isValid, isSubmitting },
    control,
  } = useForm<z.infer<typeof portSchema>>({
    resolver: zodResolver(portSchema),
    mode: "onChange",
    defaultValues: { port: defaultPort },
  });

  const currentPort = useWatch({ control, name: "port" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      toast.success("Port updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update port", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof portSchema>) => {
    await updateRuntime.mutateAsync({ environmentId, port: values.port });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Port"
        description="The port your application listens on."
        border="top"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <FormInput
            className="w-full"
            type="number"
            min={2000}
            max={54000}
            placeholder="8080"
            error={errors.port?.message}
            {...register("port", { valueAsNumber: true })}
          />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateRuntime.isLoading || isSubmitting || !isValid || currentPort === defaultPort
            }
            loading={updateRuntime.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

const CommandCard: React.FC<Props & { defaultCommand: string }> = ({
  environmentId,
  defaultCommand,
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting },
    control,
  } = useForm<z.infer<typeof commandSchema>>({
    resolver: zodResolver(commandSchema),
    mode: "onChange",
    defaultValues: { command: defaultCommand },
  });

  const currentCommand = useWatch({ control, name: "command" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      toast.success("Command updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update command", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof commandSchema>) => {
    const trimmed = values.command.trim();
    const command = trimmed === "" ? [] : trimmed.split(/\s+/).filter(Boolean);
    await updateRuntime.mutateAsync({ environmentId, command });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Command"
        description="The command to start your application."
        border="default"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <FormInput className="w-full" placeholder="npm start" {...register("command")} />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateRuntime.isLoading ||
              isSubmitting ||
              !isValid ||
              currentCommand === defaultCommand
            }
            loading={updateRuntime.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

const HealthcheckCard: React.FC<Props & { defaultMethod: "GET" | "POST"; defaultPath: string }> = ({
  environmentId,
  defaultMethod,
  defaultPath,
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting },
    setValue,
    control,
  } = useForm<z.infer<typeof healthcheckSchema>>({
    resolver: zodResolver(healthcheckSchema),
    mode: "onChange",
    defaultValues: { method: defaultMethod, path: defaultPath },
  });

  const currentMethod = useWatch({ control, name: "method" });
  const currentPath = useWatch({ control, name: "path" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      toast.success("Healthcheck updated");
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      toast.error("Failed to update healthcheck", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof healthcheckSchema>) => {
    const path = values.path.trim();
    await updateRuntime.mutateAsync({
      environmentId,
      healthcheck:
        path === ""
          ? null
          : {
              method: values.method,
              path,
              intervalSeconds: 10,
              timeoutSeconds: 5,
              failureThreshold: 3,
              initialDelaySeconds: 0,
            },
    });
  };

  const hasChanged = currentMethod !== defaultMethod || currentPath !== defaultPath;

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Healthcheck"
        description="The healthcheck endpoint for your application."
        border="bottom"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <Select
            value={currentMethod}
            onValueChange={(v) => {
              setValue("method", v as "GET" | "POST", { shouldValidate: true });
            }}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="GET">GET</SelectItem>
              <SelectItem value="POST">POST</SelectItem>
            </SelectContent>
          </Select>
          <FormInput
            className="flex grow w-full"
            placeholder="/health"
            {...register("path")}
          />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={updateRuntime.isLoading || isSubmitting || !isValid || !hasChanged}
            loading={updateRuntime.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};

export const RuntimeApplicationSettings: React.FC<Props> = ({ environmentId }) => {
  const { data } = trpc.deploy.environmentSettings.get.useQuery({ environmentId });
  const runtimeSettings = data?.runtimeSettings;

  return (
    <div>
      <PortCard environmentId={environmentId} defaultPort={runtimeSettings?.port ?? 8080} />
      <CommandCard
        environmentId={environmentId}
        defaultCommand={((runtimeSettings?.command as string[]) ?? []).join(" ")}
      />
      <HealthcheckCard
        environmentId={environmentId}
        defaultMethod={
          (runtimeSettings?.healthcheck as { method: "GET" | "POST" } | null)?.method ?? "GET"
        }
        defaultPath={(runtimeSettings?.healthcheck as { path: string } | null)?.path ?? ""}
      />
    </div>
  );
};
