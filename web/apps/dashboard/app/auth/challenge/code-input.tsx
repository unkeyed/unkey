"use client";

import { cn } from "@/lib/utils";
import { OTPInput, type SlotProps } from "input-otp";

export function CodeInput({
  value,
  onChange,
  onComplete,
  disabled,
}: {
  value: string;
  onChange: (value: string) => void;
  onComplete: (value: string) => void;
  disabled?: boolean;
}) {
  return (
    <OTPInput
      data-1p-ignore
      className="[&_input]:text-white!"
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
      "relative w-10 h-12 text-[2rem] border border-white/20 rounded-lg text-white font-light text-base",
      "flex items-center justify-center",
      "transition-all duration-300",
      "group-hover:border-white/50 group-focus-within:border-white/50",
      "outline-solid outline-0 outline-white",
      { "outline-1": props.isActive },
    )}
  >
    {props.char !== null && <div>{props.char}</div>}
  </div>
);
