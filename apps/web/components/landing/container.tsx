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
  return (
    <Component className={clsx("mx-auto max-w-7xl px-6 lg:px-8", className)}>
      <div className="max-w-2xl mx-auto lg:max-w-none">{children}</div>
    </Component>
  );
}
