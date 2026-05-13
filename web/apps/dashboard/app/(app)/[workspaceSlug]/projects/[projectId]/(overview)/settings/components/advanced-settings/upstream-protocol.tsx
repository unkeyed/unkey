"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Connections } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import { SettingField } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const PROTOCOLS = [
  { value: "http1", label: "HTTP/1.1" },
  { value: "h2c", label: "HTTP/2 (h2c)" },
] as const;

const schema = z.object({
  upstreamProtocol: z.enum(["http1", "h2c"]),
});

const displayLabel = (value: string) => PROTOCOLS.find((p) => p.value === value)?.label ?? value;

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
      icon={<Connections className="text-gray-12" iconSize="xl-medium" />}
      title="Upstream Protocol"
      description={
        <>
          Protocol used to connect to your application. If you don&apos;t know what this is, use
          HTTP/1.1.{" "}
          <a
            href="https://www.unkey.com/docs/platform/apps/settings#upstream-protocol"
            target="_blank"
            rel="noopener noreferrer"
            className="underline text-accent-11"
          >
            Learn more
          </a>
          .
        </>
      }
      displayValue={displayLabel(defaultValue)}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingField>
        <Controller
          control={control}
          name="upstreamProtocol"
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange}>
              <SelectTrigger>
                <SelectValue placeholder="Select protocol" />
              </SelectTrigger>
              <SelectContent>
                {PROTOCOLS.map((protocol) => (
                  <SelectItem
                    key={protocol.value}
                    value={protocol.value}
                    className="focus:bg-gray-3"
                  >
                    {protocol.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
      </SettingField>
    </FormSettingCard>
  );
};
