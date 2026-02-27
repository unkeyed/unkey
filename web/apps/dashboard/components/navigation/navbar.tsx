import { Dots } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import React from "react";
import { UserButton } from "./sidebar/user-button";

type BaseProps = React.PropsWithChildren<React.HTMLAttributes<HTMLElement>>;

/** Props for breadcrumb link components */
type LinkProps = {
  /** The URL that the breadcrumb links to */
  href: string;
  /** Indicates if this breadcrumb represents the current page */
  active?: boolean;
  /** If true, applies monospace font styling (useful for dynamic params like  -[namespaceId]- ) */
  isIdentifier?: boolean;
  /** Indicates if this is the last item in the breadcrumb trail */
  isLast?: boolean;
  /** Additional CSS classes to apply to the link */
  className?: string;
  /** If true, prevents navigation when clicked */
  noop?: boolean;
  children: React.ReactNode;
};

type BreadcrumbsComponent = React.ForwardRefExoticComponent<
  BaseProps &
    React.RefAttributes<HTMLElement> & {
      /** Icon to display at the start of the breadcrumb trail */
      icon: React.ReactNode;
    }
> & {
  Link: React.ForwardRefExoticComponent<LinkProps & React.RefAttributes<HTMLAnchorElement>>;
  Ellipsis: React.ForwardRefExoticComponent<React.HTMLAttributes<HTMLLIElement>>;
};

interface GlobalNavbarComponent
  extends React.ForwardRefExoticComponent<BaseProps & React.RefAttributes<HTMLElement>> {
  Actions: React.ForwardRefExoticComponent<BaseProps & React.RefAttributes<HTMLDivElement>>;
  Breadcrumbs: BreadcrumbsComponent;
  User: React.ForwardRefExoticComponent<BaseProps & React.RefAttributes<HTMLDivElement>>;
}

const NavbarActions = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex items-center gap-3", className)} {...props}>
      {props.children}
    </div>
  ),
);
NavbarActions.displayName = "GlobalNavbar.Actions";

const NavbarUser = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("hidden md:flex items-center ml-3", className)} {...props}>
      <UserButton />
    </div>
  ),
);
NavbarUser.displayName = "GlobalNavbar.User";

const BreadcrumbsLink = React.forwardRef<HTMLAnchorElement, LinkProps>(
  ({ children, href, className, active, isIdentifier: dynamic, isLast, noop, ...props }, ref) => (
    <li className="flex items-center gap-3">
      {noop ? (
        <span
          className={cn(
            "text-sm",
            active ? "text-accent-12" : "text-accent-10",
            dynamic && "font-mono",
            className,
          )}
        >
          {children}
        </span>
      ) : (
        <Link
          ref={ref}
          href={href}
          className={cn(
            "text-sm transition-colors",
            active ? "text-accent-12" : "text-accent-10 hover:text-accent-11",
            dynamic && "font-mono",
            className,
          )}
          {...(active || isLast ? { "aria-current": "page" } : {})}
          {...props}
        >
          {children}
        </Link>
      )}
      {!isLast && (
        <div className="text-accent-10" aria-hidden="true">
          /
        </div>
      )}
    </li>
  ),
);
BreadcrumbsLink.displayName = "GlobalNavbar.Breadcrumbs.Link";

const BreadcrumbsEllipsis = React.forwardRef<HTMLLIElement, React.HTMLAttributes<HTMLLIElement>>(
  ({ className, ...props }, ref) => (
    <li ref={ref} className={cn("flex gap-3 items-end", className)} {...props}>
      <span className="text-sm text-accent-10" aria-label="More pages">
        <Dots className="h-4 w-4 text-accent-10" />
      </span>
      <div className="text-accent-10" aria-hidden="true">
        /
      </div>
    </li>
  ),
);
BreadcrumbsEllipsis.displayName = "GlobalNavbar.Breadcrumbs.Ellipsis";

const Breadcrumbs = React.forwardRef<HTMLElement, BaseProps & { icon: React.ReactNode }>(
  ({ children, className, icon, ...props }, ref) => {
    const childrenArray = React.Children.toArray(children);
    return (
      <nav ref={ref} aria-label="breadcrumb" className={cn("flex", className)} {...props}>
        <ol className="flex items-center gap-3">
          <li>
            <Button
              variant="outline"
              className="size-6 p-0 [&>svg]:size-[18px] bg-gray-4 hover:bg-gray-5"
            >
              {icon}
            </Button>
          </li>
          {childrenArray.map((child, index) => {
            if (!React.isValidElement(child)) {
              return null;
            }
            if (child.type === BreadcrumbsLink) {
              const childProps = child.props;
              if (typeof childProps === "object" && childProps !== null) {
                const enhancedProps = {
                  ...childProps,
                  isLast: index === childrenArray.length - 1,
                  key: child.key || `breadcrumb-${index}`,
                };
                return React.cloneElement(child, enhancedProps);
              }
            }

            // biome-ignore lint/suspicious/noArrayIndexKey: Usage of index is acceptable here.
            return React.cloneElement(child, { key: index });
          })}
        </ol>
      </nav>
    );
  },
) as BreadcrumbsComponent;
Breadcrumbs.displayName = "GlobalNavbar.Breadcrumbs";
Breadcrumbs.Link = BreadcrumbsLink;
Breadcrumbs.Ellipsis = BreadcrumbsEllipsis;

export const Navbar = React.forwardRef<HTMLElement, BaseProps>(
  ({ children, className, ...props }, ref) => {
    const childrenArray = React.Children.toArray(children);
    const breadcrumbs = childrenArray.find(
      (child) => React.isValidElement(child) && child.type === Breadcrumbs,
    );
    const actions = childrenArray.find(
      (child) => React.isValidElement(child) && child.type === NavbarActions,
    );
    const user = childrenArray.find(
      (child) => React.isValidElement(child) && child.type === NavbarUser,
    );

    return (
      <nav
        ref={ref}
        className={cn(
          "w-full p-4 border-b border-gray-4 bg-transparent flex items-center min-h-[65px]",
          className,
        )}
        {...props}
      >
        {breadcrumbs}
        <div className="flex-1" />
        {actions}
        {user || <NavbarUser />}
      </nav>
    );
  },
) as GlobalNavbarComponent;
Navbar.displayName = "GlobalNavbar";

Navbar.Actions = NavbarActions;
Navbar.User = NavbarUser;
Navbar.Breadcrumbs = Breadcrumbs;
