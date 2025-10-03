"use client";
import { Code } from "@unkey/icons";
import { Button, FormTextarea, toast } from "@unkey/ui";
import { useFormContext, useWatch } from "react-hook-form";
import type { MetadataFormValues } from "../create-key.schema";
import { ProtectionSwitch } from "./protection-switch";

export const EXAMPLE_JSON = {
  user: {
    id: "user_123456",
    role: "admin",
    permissions: ["read", "write", "delete"],
  },
};

export const MetadataSetup = ({
  overrideEnabled = false,
}: {
  overrideEnabled?: boolean;
}) => {
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
  });

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

  return (
    <div className="space-y-5 px-2 py-1">
      {!overrideEnabled && (
        <ProtectionSwitch
          description="Add custom metadata to your API key as a JSON object. This metadata will be available when verifying the key."
          title="Metadata"
          icon={<Code className="text-gray-12" iconsize="sm-regular" />}
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
          description="Add structured JSON data to this key. Must be valid JSON format."
          error={errors.metadata?.data?.message}
          disabled={!metadataEnabled}
          readOnly={!metadataEnabled}
          rows={15}
          {...register("metadata.data", {
            validate: (value) => {
              if (metadataEnabled && (!value || !validateJSON(value))) {
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
