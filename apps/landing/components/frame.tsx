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
        "flex max-w-7xl w-fit mx-auto rounded-[30.5px] bg-gradient-to-b from-white/0 to-white/10 p-[.75px] overflow-hidden",
        className,
      )}
    >
      <div
        className={cn(
          size === "lg" && "p-2",
          size === "md" && "p-1",
          size === "sm" && "p-[3px]",
          "bg-gradient-to-r from-white/10 to-white/20 w-full",
        )}
      >
        <div className="rounded-[24px] overflow-hidden bg-gradient-to-b from-white/20 to-white/10 p-[.75px] w-full h-full">
          {children}
        </div>
      </div>
    </Component>
  );
  // background: conic-gradient(from 0deg at 58.39% 29.49%, rgba(255, 255, 255, 0) 0deg, #FFFFFF 0.04deg, rgba(255, 255, 255, 0) 60deg, rgba(255, 255, 255, 0) 360deg),
  // linear-gradient(180deg, rgba(255, 255, 255, 0.06) 0%, rgba(255, 255, 255, 0.08) 100%),
  // radial-gradient(57.42% 100% at 100% 0%, rgba(255, 255, 255, 0.25) 0%, rgba(255, 255, 255, 0.2) 50%, rgba(255, 255, 255, 0.1) 100%) /* warning: gradient uses a rotation that is not supported by CSS and may not behave as expected */;
}
