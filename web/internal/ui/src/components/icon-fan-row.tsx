import { Children } from "react";
import type * as React from "react";
import { cn } from "../lib/utils";

const OPACITY_BY_DISTANCE_FROM_CENTER = ["opacity-90", "opacity-75", "opacity-50"];

function IconFanRow({ className, children, ...props }: Omit<React.ComponentProps<"div">, "style">) {
  const icons = Children.toArray(children);
  const centerIndex = Math.floor(icons.length / 2);
  return (
    <div
      aria-hidden="true"
      className={cn("p-2 mb-5", className)}
      style={{
        maskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
        WebkitMaskImage:
          "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
      }}
      {...props}
    >
      <div className="flex gap-6 items-center justify-center text-gray-12">
        {icons.map((icon, index) => {
          const distanceFromCenter = Math.min(
            Math.abs(index - centerIndex),
            OPACITY_BY_DISTANCE_FROM_CENTER.length - 1,
          );
          return (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: static decorative row, index is stable
              key={index}
              className={cn(
                "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none",
                index === centerIndex
                  ? "size-16 [&_svg]:size-9 [&_svg_[stroke-width]]:[stroke-width:0.75]"
                  : "size-9 [&_svg]:size-[18px]",
                OPACITY_BY_DISTANCE_FROM_CENTER[distanceFromCenter],
              )}
            >
              {icon}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export { IconFanRow };
