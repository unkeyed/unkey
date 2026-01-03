import Link from "next/link";

export const NavLink = ({
  href,
  external,
  onClick,
  children,
  isLoadMoreButton = false,
}: {
  href: string;
  external?: boolean;
  onClick?: () => void;
  children: React.ReactNode;
  isLoadMoreButton?: boolean;
}) => {
  // For the load more button, we use a button instead of a link
  if (isLoadMoreButton) {
    return (
      <button type="button" onClick={onClick} className="w-full text-left">
        {children}
      </button>
    );
  }

  return (
    <Link
      prefetch={!external}
      href={href}
      onClick={onClick}
      target={external ? "_blank" : undefined}
    >
      {children}
    </Link>
  );
};
