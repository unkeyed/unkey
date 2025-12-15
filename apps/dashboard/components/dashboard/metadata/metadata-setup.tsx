"use client";
import type { MetadataFormValues } from "@/lib/schemas/metadata";
import { Code } from "@unkey/icons";
import { Button, FormTextarea, toast } from "@unkey/ui";
import { useFormContext, useWatch } from "react-hook-form";
import { ProtectionSwitch } from "./protection-switch";

export const EXAMPLE_JSON = {
  user: {
    id: "user_123456",
    role: "admin",
    permissions: ["read", "write", "delete"],
  },
};

type EntityType = "key" | "identity";

interface MetadataSetupProps {
  overrideEnabled?: boolean;
  entityType: EntityType;
}

const ENTITY_DESCRIPTIONS: Record<
  EntityType,
  {
    switch: string;
    textarea: string;
  }
> = {
  key: {
    switch:
      "Add custom metadata to your API key as a JSON object. This metadata will be available when verifying the key.",
    textarea: "Add structured JSON data to this key. Must be valid JSON format.",
  },
  identity: {
    switch:
      "Add custom metadata to this identity as a JSON object. This metadata will be available when verifying keys associated with this identity.",
    textarea: "Add structured JSON data to this identity. Must be valid JSON format.",
  },
};

export const MetadataSetup = ({ overrideEnabled = false, entityType }: MetadataSetupProps) => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
    trigger,
  } = useFormContext<MetadataFormValues>();

  const metadataEnabled = useWatch({
    control,
    name: "metadata.enabled",
  });

  const currentMetadata = useWatch({
    control,
    name: "metadata.data",
  }) as string | undefined;

  const handleSwitchChange = (checked: boolean) => {
    setValue("metadata.enabled", checked);
    // Only set example json if its first time
    if (checked && !currentMetadata) {
      setValue("metadata.data", JSON.stringify(EXAMPLE_JSON, null, 2));
    }

    trigger("metadata");
  };

  const formatJSON = () => {
    try {
      const parsed = JSON.parse(currentMetadata || "{}");
      setValue("metadata.data", JSON.stringify(parsed, null, 2));
    } catch (error) {
      if (error instanceof Error) {
        toast.error(error.message);
      } else {
        toast.error("Please check your JSON syntax");
      }
    }
  };

  const validateJSON = (jsonString: string): boolean => {
    try {
      JSON.parse(jsonString);
      return true;
    } catch {
      return false;
    }
  };

  const descriptions = ENTITY_DESCRIPTIONS[entityType];

  return (
    <div className="space-y-5 px-2 py-1">
      {!overrideEnabled && (
        <ProtectionSwitch
          description={descriptions.switch}
          title="Metadata"
          icon={<Code className="text-gray-12" iconSize="sm-regular" />}
          checked={metadataEnabled}
          onCheckedChange={handleSwitchChange}
          {...register("metadata.enabled")}
        />
      )}
      <div className="space-y-2 h-fit duration-300">
        <FormTextarea
          placeholder={JSON.stringify(EXAMPLE_JSON, null, 2)}
          label="Metadata"
          className="[&_textarea:first-of-type]:font-mono h-full"
          rightIcon={
            <Button
              size="sm"
              variant="outline"
              onClick={formatJSON}
              disabled={!metadataEnabled || Boolean(errors.metadata?.data?.message)}
              type="button"
            >
              <div className="text-[13px]">Format</div>
            </Button>
          }
          description={descriptions.textarea}
          error={errors.metadata?.data?.message}
          disabled={!metadataEnabled}
          readOnly={!metadataEnabled}
          rows={15}
          {...register("metadata.data", {
            validate: (value) => {
              if (metadataEnabled && (!value || !validateJSON(value as string))) {
                return "Must be valid JSON";
              }
              return true;
            },
          })}
        />
      </div>
    </div>
  );
};
