import clsx from "clsx";

export function Border({
  className,
  position = "top",
  invert = false,
  as: Component = "div",
  ...props
}: {
  className?: string;
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
      )}
      {...props}
    />
  );
}
