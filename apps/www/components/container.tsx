import clsx from "clsx";

export function Container({
  as: Component = "div",
  className,
  children,
}: {
  as?: any;
  className?: string;
  children: React.ReactNode;
}) {
  return <Component className={clsx("container", className)}>{children}</Component>;
}
