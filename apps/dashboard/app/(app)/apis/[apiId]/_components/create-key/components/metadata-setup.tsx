"use client";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { Code } from "@unkey/icons";
import { FormTextarea } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import type { FormValues } from "../schema";

export type MetadataFormValues = Pick<FormValues, "metadata">;

const exampleJSON = {
  user: {
    id: "user_123456",
    role: "admin",
    permissions: ["read", "write", "delete"],
  },
  environment: "production",
  limits: {
    requestsPerMonth: 10000,
    dataStorage: "5GB",
  },
  created: "2025-04-15T10:30:00Z",
  tags: ["api", "backend", "critical"],
};

export const MetadataSetup = () => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
    getValues,
    trigger,
  } = useFormContext<FormValues>();

  const [lastMetadata, setLastMetadata] = useState<string | null>(null);

  const metadataEnabled = useWatch({
    control,
    name: "metadata.enabled",
    defaultValue: false,
  });

  const currentMetadata = useWatch({
    control,
    name: "metadata.data",
  });

  // Update last metadata when content changes while enabled
  useEffect(() => {
    if (metadataEnabled && currentMetadata) {
      setLastMetadata(currentMetadata);
    }
  }, [metadataEnabled, currentMetadata]);

  const handleSwitchChange = (checked: boolean) => {
    setValue("metadata.enabled", checked);
    if (checked) {
      if (lastMetadata) {
        setValue("metadata.data", lastMetadata);
      } else if (!getValues("metadata.data")) {
        setValue("metadata.data", "{}");
      }
    }
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

  const loadExample = () => {
    setValue("metadata.data", JSON.stringify(exampleJSON, null, 2));
    trigger("metadata.data");
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
      <div className="flex flex-row py-5 pl-5 pr-[26px] gap-14 justify-between border rounded-xl border-grayA-5 bg-white dark:bg-black items-center">
        <div className="flex flex-col gap-4">
          <div className="flex gap-3">
            <div className="p-1.5 bg-grayA-3 rounded-md border border-grayA-3">
              <Code className="text-gray-12" size="sm-regular" />
            </div>
            <div className="text-sm font-medium text-gray-12">Metadata</div>
          </div>
          <div className="text-gray-9 text-xs">
            Add custom metadata to your API key as a JSON object. This metadata will be available
            when verifying the key.
          </div>
        </div>
        <Switch
          checked={metadataEnabled}
          onCheckedChange={handleSwitchChange}
          className="
            h-4 w-7
            data-[state=checked]:bg-success-9
            data-[state=checked]:ring-2
            data-[state=checked]:ring-successA-5
            data-[state=unchecked]:bg-gray-3
            data-[state=unchecked]:ring-2
            data-[state=unchecked]:ring-grayA-3
            [&>span]:h-3.5 [&>span]:w-3.5
          "
          {...register("metadata.enabled")}
        />
      </div>

      <div
        className="space-y-2 h-fit transition-opacity duration-300"
        style={{ opacity: metadataEnabled ? 1 : 0.5 }}
      >
        <div className="flex justify-end mb-2">
          <Button
            size="sm"
            variant="outline"
            onClick={loadExample}
            disabled={!metadataEnabled}
            type="button"
            className="mr-2"
          >
            Load Example
          </Button>
        </div>

        <FormTextarea
          placeholder='{ "example": "value" }'
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
