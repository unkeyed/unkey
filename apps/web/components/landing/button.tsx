import clsx from "clsx";
import Link from "next/link";

export function Button({
  invert,
  href,
  className,
  children,
  ...props
}: {
  invert?: boolean;
  href?: string;
  className?: string;
  children?: React.ReactNode;
}) {
  className = clsx(
    className,
    "inline-flex rounded-full px-4 py-1.5 text-sm font-semibold transition",
    invert
      ? "bg-white text-gray-950 hover:bg-gray-200"
      : "bg-gray-950 text-white hover:bg-gray-800",
  );

  const inner = <span className="relative top-px">{children}</span>;

  if (href) {
    return (
      <Link href={href} className={className} {...props}>
        {inner}
      </Link>
    );
  }

  return (
    <button className={className} {...props} type="button">
      {inner}
    </button>
  );
}
