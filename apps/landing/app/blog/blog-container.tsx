import clsx from "clsx";

export function BlogContainer({
  as: Component = "div",
  className,
  children,
}: {
  as?: any;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <Component className={clsx("mx-auto max-w-full sm:px-6 lg:px-8", className)}>
      <div className="max-w-[1440px] mx-auto">{children}</div>
    </Component>
  );
}
