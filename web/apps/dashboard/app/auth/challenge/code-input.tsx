"use client";

import { cn } from "@/lib/utils";
import { OTPInput, type SlotProps } from "input-otp";

export function CodeInput({
  value,
  onChange,
  onComplete,
  disabled,
  // OTP code entry always lives on a dedicated screen, so focusing the input
  // on mount is the expected UX: typing works immediately and Enter submits.
  autoFocus = true,
}: {
  value: string;
  onChange: (value: string) => void;
  onComplete: (value: string) => void;
  disabled?: boolean;
  autoFocus?: boolean;
}) {
  return (
    <OTPInput
      data-1p-ignore
      autoFocus={autoFocus}
      className="[&_input]:text-gray-12!"
      value={value}
      onChange={onChange}
      onComplete={onComplete}
      disabled={disabled}
      maxLength={6}
      render={({ slots }) => (
        <div className="flex items-center justify-between">
          {slots.slice(0, 6).map((slot, idx) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: slot order is stable
            <Slot key={idx} {...slot} />
          ))}
        </div>
      )}
    />
  );
}

const Slot: React.FC<SlotProps> = (props) => (
  <div
    className={cn(
      "relative w-10 h-12 text-[2rem] border rounded-lg font-light text-base border-gray-6 text-gray-12",
      "flex items-center justify-center",
      "transition-all duration-300",
      "group-hover:border-gray-8 group-focus-within:border-gray-8",
      "outline-solid outline-0 outline-gray-12",
      { "outline-1": props.isActive },
    )}
  >
    {props.char !== null && <div>{props.char}</div>}
  </div>
);
