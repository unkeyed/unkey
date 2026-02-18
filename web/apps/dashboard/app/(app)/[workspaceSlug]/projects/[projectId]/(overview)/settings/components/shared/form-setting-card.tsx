import { Button, SettingCard, type SettingCardBorder } from "@unkey/ui";
import type React from "react";
import { SelectedConfig } from "./selected-config";

type EditableSettingCardProps = {
  icon: React.ReactNode;
  title: string;
  description: string;
  border?: SettingCardBorder;

  displayValue: React.ReactNode;

  onSubmit: React.FormEventHandler<HTMLFormElement>;
  children: React.ReactNode;

  canSave: boolean;
  isSaving: boolean;
};

export const FormSettingCard = ({
  icon,
  title,
  description,
  border,
  displayValue,
  onSubmit,
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
        <form className="px-4 py-4 flex flex-col gap-3 bg-grayA-2 rounded-b-xl" onSubmit={onSubmit}>
          {children}
          <div className="flex justify-end">
            <Button
              type="submit"
              variant="primary"
              className="px-3 py-3"
              size="sm"
              disabled={!canSave}
              loading={isSaving}
            >
              Save
            </Button>
          </div>
        </form>
      }
    >
      <SelectedConfig label={displayValue} />
    </SettingCard>
  );
};
