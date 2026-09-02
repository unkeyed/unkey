"use client";
import type { MetadataFormValues } from "@/lib/schemas/metadata";
import {
  Button,
  FormField,
  InputGroup,
  InputGroupAddon,
  InputGroupTextarea,
  toast,
} from "@unkey/ui";
import { IconCodeOutline18 } from "nucleo-ui-outline-18";
import { useController, useFormContext, useWatch } from "react-hook-form";
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

  // Keep the discriminator registered independently of portal-mounted DOM refs.
  const {
    field: { value: metadataEnabled },
  } = useController({
    control,
    name: "metadata.enabled",
  });

  const currentMetadata = useWatch({
    control,
    name: "metadata.data",
  }) as string | undefined;

  const handleSwitchChange = (checked: boolean) => {
    if (checked) {
      setValue("metadata", {
        enabled: true,
        data: currentMetadata || JSON.stringify(EXAMPLE_JSON, null, 2),
      });
    } else {
      setValue("metadata", { enabled: false });
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
    <div className="flex flex-col gap-5 px-2 py-1">
      {!overrideEnabled && (
        <ProtectionSwitch
          description={descriptions.switch}
          title="Metadata"
          icon={<IconCodeOutline18 className="text-gray-12" />}
          checked={metadataEnabled}
          onCheckedChange={handleSwitchChange}
        />
      )}
      <div className="flex flex-col gap-2 h-fit duration-300">
        <FormField
          className="[&_textarea:first-of-type]:font-mono h-full"
          label="Metadata"
          description={descriptions.textarea}
          error={errors.metadata?.data?.message}
        >
          {(field) => (
            <InputGroup variant={field.variant} className="items-start">
              <InputGroupTextarea
                id={field.id}
                placeholder={JSON.stringify(EXAMPLE_JSON, null, 2)}
                aria-describedby={field.describedBy}
                aria-invalid={field.invalid}
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
              <InputGroupAddon align="inline-end" className="pt-3">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={formatJSON}
                  disabled={!metadataEnabled || field.invalid}
                  type="button"
                >
                  <div className="text-[13px]">Format</div>
                </Button>
              </InputGroupAddon>
            </InputGroup>
          )}
        </FormField>
      </div>
    </div>
  );
};
