import { cn } from "@/lib/utils";
import { Button, InfoTooltip, SettingCard, type SettingCardBorder } from "@unkey/ui";
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

  saveState: SaveState;

  ref?: React.Ref<HTMLFormElement>;
  className?: string;
  autoSave?: boolean;
};

export const FormSettingCard = ({
  icon,
  title,
  description,
  border,
  displayValue,
  onSubmit,
  children,
  saveState,
  ref,
  className,
  autoSave,
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
            e.preventDefault();
            onSubmit(e);
          }}
          onBlur={(e) => {
            if (!autoSave || saveState.status !== "ready") {
              return;
            }
            const relatedTarget = e.relatedTarget instanceof Node ? e.relatedTarget : null;
            if (!e.currentTarget.contains(relatedTarget)) {
              e.currentTarget.requestSubmit();
            }
          }}
        >
          <div
            className={cn(
              "px-4 pt-4 flex flex-col gap-0.5 overflow-y-auto max-h-[500px]",
              autoSave ? "pb-4" : "pb-2",
            )}
          >
            {children}
          </div>
          {!autoSave && (
            <div className="px-4 pt-2 pb-4 flex justify-end">
              <InfoTooltip
                content={saveState.status === "disabled" ? saveState.reason : undefined}
                disabled={
                  saveState.status !== "disabled" || !("reason" in saveState && saveState.reason)
                }
                asChild
                variant="inverted"
              >
                <Button
                  type="submit"
                  variant="primary"
                  className="px-3 py-3"
                  size="sm"
                  disabled={saveState.status !== "ready"}
                  loading={saveState.status === "saving"}
                >
                  Save
                </Button>
              </InfoTooltip>
            </div>
          )}
        </form>
      }
    >
      <SelectedConfig
        label={displayValue ?? <span className="text-gray-11 font-normal">None</span>}
      />
    </SettingCard>
  );
};

export type SaveState =
  | { status: "ready" }
  | { status: "disabled"; reason?: string }
  | { status: "saving" };

export function resolveSaveState(checks: ReadonlyArray<[boolean, SaveState]>): SaveState {
  for (const [condition, state] of checks) {
    if (condition) {
      return state;
    }
  }
  return { status: "ready" };
}
