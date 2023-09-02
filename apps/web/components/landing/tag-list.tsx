import clsx from "clsx";

export function TagList({
  className,
  children,
}: {
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <ul role="list" className={clsx(className, "flex flex-wrap gap-4")}>
      {children}
    </ul>
  );
}

export function TagListItem({
  className,
  children,
}: {
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <li className={clsx("rounded-full bg-gray-100 px-4 py-1.5 text-base text-gray-600", className)}>
      {children}
    </li>
  );
}
