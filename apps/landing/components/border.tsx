import clsx from "clsx";

export function Border({
  className,
  position = "top",
  invert = false,
  as: Component = "div",
  ...props
}: {
  className?: string;
  position?: "top" | "left";
  invert?: boolean;
  as?: any;
  [key: string]: any;
}) {
  return (
    <Component
      className={clsx(
        className,
        "relative before:absolute after:absolute",
        invert ? "before:bg-white after:bg-white/10" : "before:bg-gray-950 after:bg-gray-950/10",
        position === "top" &&
          "before:left-0 before:top-0 before:h-px before:w-6 after:left-8 after:right-0 after:top-0 after:h-px",
        position === "left" &&
          "before:left-0 before:top-0 before:h-6 before:w-px after:bottom-0 after:left-0 after:top-8 after:w-px",
      )}
      {...props}
    />
  );
}
