import * as SliderPrimitive from "@radix-ui/react-slider";
import * as React from "react";
import { cn } from "../lib/utils";

type SliderProps = React.ComponentPropsWithoutRef<typeof SliderPrimitive.Root> & {
  rangeClassName?: string;
  rangeStyle?: React.CSSProperties;
};

const Slider = React.forwardRef<React.ComponentRef<typeof SliderPrimitive.Root>, SliderProps>(
  ({ className, rangeClassName, rangeStyle, ...props }, ref) => (
    <SliderPrimitive.Root
      ref={ref}
      className={cn("relative flex w-full touch-none select-none items-center", className)}
      {...props}
    >
      <SliderPrimitive.Track className="relative h-1.5 w-full grow overflow-hidden rounded-full bg-grayA-3">
        <SliderPrimitive.Range
          className={cn("absolute h-full bg-accent-12", rangeClassName)}
          style={rangeStyle}
        />
      </SliderPrimitive.Track>
      <SliderPrimitive.Thumb className="block h-4 w-4 rounded-full border border-grayA-6 bg-gray-2 shadow transition-colors duration-300 hover:border-grayA-8 focus:ring focus:ring-gray-5 focus-visible:outline-none disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50" />
    </SliderPrimitive.Root>
  ),
);
Slider.displayName = SliderPrimitive.Root.displayName;

export { Slider };
