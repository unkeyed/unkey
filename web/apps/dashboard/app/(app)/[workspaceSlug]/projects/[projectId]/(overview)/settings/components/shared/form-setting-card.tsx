import { Button, SettingCard, type SettingCardBorder } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
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

  ref?: React.Ref<HTMLFormElement>;
  className?: string;
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
  ref,
  className,
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
        <form
          className={cn("px-4 py-4 flex flex-col gap-3 bg-grayA-2 rounded-b-xl", className)}
          ref={ref}
          onSubmit={onSubmit}
        >
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
