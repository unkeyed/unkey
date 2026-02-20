import { cn } from "@/lib/utils";
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
          className={cn("flex flex-col bg-grayA-2 rounded-b-xl", className)}
          ref={ref}
          onSubmit={(e) => {
            //Without this form will toggle the chevron and collapse the section
            e.preventDefault()
            onSubmit(e)
          }}
        >
          <div className="px-4 pt-4 pb-2 flex flex-col gap-3 overflow-y-auto max-h-[500px]">
            {children}
          </div>
          <div className="px-4 pt-2 pb-4 flex justify-end">
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
