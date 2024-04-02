import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";
import { SparkleIcon } from "./svg/template-page";
import { HoverBorderGradient } from "./ui/hover-border-gradient";

type Props = {
  className?: string;
  IconLeft?: LucideIcon;
  label: string;
  IconRight?: LucideIcon;
  onClick?: React.ChangeEventHandler;
};

export const PrimaryButton: React.FC<Props> = ({
  className,
  IconLeft,
  label,
  IconRight,
  onClick,
}) => {
  return (
    <div className="relative group/button">
      <div className="absolute -inset-0.5 bg-white rounded-lg blur-2xl group-hover/button:opacity-30 transition duration-300  opacity-0 " />
      <button
        onClick={onClick}
        type="button"
        className={cn(
          "relative flex items-center px-4 gap-2 text-sm font-semibold text-black  bg-white group-hover:bg-white/90 duration-1000 rounded-lg h-10",
          className,
        )}
      >
        {IconLeft ? <IconLeft className="w-4 h-4" /> : null}
        {label}
        {IconRight ? <IconRight className="w-4 h-4" /> : null}
      </button>
    </div>
  );
};

export const SecondaryButton: React.FC<Props> = ({ className, IconLeft, label, IconRight }) => {
  return (
    <button
      type="button"
      className={cn(
        "items-center gap-2 px-4 duration-500 text-white/50 hover:text-white h-10 hidden sm:flex",
        className,
      )}
    >
      {IconLeft ? <IconLeft className="w-4 h-4" /> : null}
      {label}
      {IconRight ? <IconRight className="w-4 h-4" /> : null}
    </button>
  );
};

export const RainbowDarkButton: React.FC<Props> = ({ className, label, IconRight }) => {
  return (
    <div className={cn("p-[.75px] w-fit mx-auto", className)}>
      <HoverBorderGradient
        containerClassName="rounded-full"
        as="button"
        className="bg-black text-white flex items-center space-x-2"
      >
        <SparkleIcon className="text-white mr-2" />
        {label}
        {IconRight ? <IconRight className="w-4 h-4" /> : null}
      </HoverBorderGradient>
    </div>
  );
};

export function HoverBorderGradientButton() {
  return (
    <div className="m-40 flex justify-center text-center">
      <HoverBorderGradient
        containerClassName="rounded-full"
        as="button"
        className="bg-black text-white flex items-center space-x-2"
      >
        <span>Aceternity UI</span>
      </HoverBorderGradient>
    </div>
  );
}
