import { Button, SettingCard } from "@unkey/ui";
import type React from "react";
import { SelectedConfig } from "./selected-config";

type EditableSettingCardProps = {
  icon: React.ReactNode;
  title: string;
  description: string;
  border?: "top" | "bottom" | "both" | "none" | "default";

  displayValue: React.ReactNode;

  formId: string;
  children: React.ReactNode;

  canSave: boolean;
  isSaving: boolean;
};

export const EditableSettingCard = ({
  icon,
  title,
  description,
  border,
  displayValue,
  formId,
  children,
  canSave,
  isSaving,
}: EditableSettingCardProps) => {
  return (
    <SettingCard
      className="px-4 py-[18px]"
      icon={icon}
      title={title}
      description={description}
      border={border}
      contentWidth="w-full lg:w-[320px] justify-end"
      expandable={
        <div className="px-4 py-4 flex flex-col gap-3 bg-grayA-2 rounded-b-xl">
          {children}
          <div className="flex justify-end">
            <Button
              type="submit"
              form={formId}
              variant="primary"
              className="px-3 py-3"
              size="sm"
              disabled={!canSave}
              loading={isSaving}
            >
              Save
            </Button>
          </div>
        </div>
      }
    >
      <SelectedConfig label={displayValue} />
    </SettingCard>
  );
};
