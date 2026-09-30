"use client";

import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";
import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const PROTOCOLS = [
  { value: "http1", label: "HTTP/1.1", hint: "Works with every server." },
  { value: "h2c", label: "HTTP/2 (h2c)", hint: "Cleartext HTTP/2. For gRPC and streaming." },
] as const;

const schema = z.object({
  upstreamProtocol: z.enum(["http1", "h2c"]),
});

export const UpstreamProtocol = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { upstreamProtocol: defaultValue } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    mode: "onChange",
    defaultValues: { upstreamProtocol: defaultValue },
  });

  const currentProtocol = useWatch({ control, name: "upstreamProtocol" });

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [currentProtocol === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof schema>) => {
    updateAllEnvironments((draft) => {
      draft.upstreamProtocol = values.upstreamProtocol;
    });
  };

  return (
    <FormSettingCard
      title="Upstream protocol"
      description={
        <>
          Protocol Unkey uses to reach your app.{" "}
          <a
            href="https://www.unkey.com/docs/platform/apps/settings#upstream-protocol"
            target="_blank"
            rel="noopener noreferrer"
            className="underline text-gray-11"
          >
            Learn more
          </a>
          .
        </>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <Controller
          control={control}
          name="upstreamProtocol"
          render={({ field }) => (
            <RadioGroup
              aria-label="Upstream protocol"
              value={field.value}
              onValueChange={(value) => {
                const protocol = PROTOCOLS.find((option) => option.value === value);
                if (protocol) {
                  field.onChange(protocol.value);
                }
              }}
              className="grid gap-3 sm:grid-cols-2"
            >
              {PROTOCOLS.map((protocol) => (
                <Radio.Root
                  key={protocol.value}
                  value={protocol.value}
                  className="group flex items-start gap-3 rounded-lg border border-grayA-5 p-3.5 text-left transition-colors hover:border-grayA-7 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7 data-checked:border-gray-12 data-checked:bg-grayA-2 data-checked:ring-1 data-checked:ring-gray-12"
                >
                  <span className="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full border border-grayA-7 group-data-checked:border-gray-12 group-data-checked:bg-gray-12">
                    <Radio.Indicator className="size-1.5 rounded-full bg-gray-1" />
                  </span>
                  <span className="flex flex-col gap-1">
                    <span className="font-mono text-sm font-medium text-gray-12">
                      {protocol.label}
                    </span>
                    <span className="text-xs leading-5 text-gray-11">{protocol.hint}</span>
                  </span>
                </Radio.Root>
              ))}
            </RadioGroup>
          )}
        />
      </SettingField>
    </FormSettingCard>
  );
};
