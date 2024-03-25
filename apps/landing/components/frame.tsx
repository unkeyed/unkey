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
    <Component className={cn("bg-black relative", className)}>
      <div
        className={cn(
          "bg-gradient-to-r from-white/10 to-white/20 ",
          {
            "rounded-[36px] p-[8px]": size === "lg",
            "rounded-[24px] p-[6px]": size === "md",
            "rounded-[18px] p-[2px]": size === "sm",
          },
          className,
        )}
      >
        <div
          className={cn("overflow-hidden", {
            "rounded-[28px]": size === "lg",
            "rounded-[18px]": size === "md",
            "rounded-[16px]": size === "sm",
          })}
        >
          {children}
        </div>
      </div>
    </Component>
  );
}
