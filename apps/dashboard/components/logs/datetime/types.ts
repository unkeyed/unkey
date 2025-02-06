import type { PropsWithChildren } from "react";

export type SuggestionOption = {
  id: number;
  value: string | undefined;
  display: string;
  checked: boolean;
};

export type OptionsType = SuggestionOption[];

export interface DatetimePopoverProps extends PropsWithChildren {
  initialTitle: string;
  initialSelected: boolean;
  initialTimeValues: { startTime?: number; endTime?: number; since?: string };
  onSuggestionChange: (title: string, selected: boolean) => void;
  onDateTimeChange: (startTime?: number, endTime?: number, since?: string) => void;
}
