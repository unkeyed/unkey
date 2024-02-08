import { cn } from "@/lib/utils";

export function Frame({
  as: Component = "div",
  size,
  className,
  children,
}: {
  as?: any;
  size: "sm" | "md" | "lg";
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <Component
      className={cn(
        size === "lg" && "rounded-[30.5px]",
        size === "md" && "rounded-[24px]",
        size === "sm" && "rounded-[18px]",
        "max-w-7xl w-fit mx-auto bg-gradient-to-b from-white/0 to-white/10 p-[.75px] overflow-hidden relative z-2",
        className,
      )}
    >
      <div
        className={cn(
          size === "lg" && "p-2 rounded-[30px]",
          size === "md" && "p-1 rounded-[24px]",
          size === "sm" && "p-[1px] rounded-[18px]",
          "bg-gradient-to-r from-white/10 to-white/20 overflow-hidden",
        )}
      >
        <div
          className={cn(
            size === "lg" && "rounded-[24px]",
            size === "md" && "rounded-[24px]",
            size === "sm" && "rounded-[17px]",
            "overflow-hidden bg-gradient-to-b from-white/20 to-white/10 ",
          )}
        >
          {children}
        </div>
      </div>
    </Component>
  );
}
