import { cn } from "@/lib/utils";
import { LucideIcon } from "lucide-react";

type Props = {
  className?: string;
  IconLeft?: LucideIcon;
  label: string;
  IconRight?: LucideIcon;
};

export const PrimaryButton: React.FC<Props> = ({ className, IconLeft, label, IconRight }) => {
  return (
    <div className="relative group/button">
      <div className="absolute -inset-0.5 bg-gradient-to-r from-[#0239FC] to-[#7002FC] rounded-lg blur-md hover:opacity-75 group-hover/button:opacity-100 transition duration-1000 hover:rotate-20  opacity-0 group-hover/button:duration-200" />
      <button
        type="button"
        className={cn(
          "relative flex items-center px-4 gap-2 text-sm font-semibold text-black bg-white rounded-lg h-10",
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
        "flex items-center gap-2 px-4 duration-500 text-white/50 hover:text-white h-10",
        className,
      )}
    >
      {IconLeft ? <IconLeft className="w-4 h-4" /> : null}
      {label}
      {IconRight ? <IconRight className="w-4 h-4" /> : null}
    </button>
  );
};
