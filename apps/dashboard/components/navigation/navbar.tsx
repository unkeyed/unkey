import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { MoreHorizontal } from "lucide-react";
import React from "react";

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
}

export const Navbar = React.forwardRef<HTMLElement, BaseProps>(
  ({ children, className, ...props }, ref) => (
    <nav
      ref={ref}
      className={cn(
        "w-full p-4 border-b border-gray-4 bg-background justify-between flex",
        className,
      )}
      {...props}
    >
      {children}
    </nav>
  ),
) as GlobalNavbarComponent;

Navbar.Actions = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex items-center gap-3", className)} {...props}>
      {props.children}
    </div>
  ),
);

const Breadcrumbs = React.forwardRef<HTMLElement, BaseProps & { icon: React.ReactNode }>(
  ({ children, className, icon, ...props }, ref) => {
    const childrenArray = React.Children.toArray(children);
    return (
      <nav ref={ref} aria-label="breadcrumb" className={cn("flex", className)} {...props}>
        <ol className="flex items-center gap-3">
          <li className="mr-1">
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
            if (child.type === Breadcrumbs.Link) {
              return React.cloneElement(child, {
                ...child.props,
                isLast: index === childrenArray.length - 1,
                key: child.key || `breadcrumb-${index}`,
              });
            }

            // biome-ignore lint/suspicious/noArrayIndexKey: Usage of index is acceptable here.
            return React.cloneElement(child, { key: index });
          })}
        </ol>
      </nav>
    );
  },
) as BreadcrumbsComponent;

Breadcrumbs.Link = React.forwardRef<HTMLAnchorElement, LinkProps>(
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
        <a
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
        </a>
      )}
      {!isLast && (
        <div className="text-accent-10" aria-hidden="true">
          /
        </div>
      )}
    </li>
  ),
);

Breadcrumbs.Ellipsis = React.forwardRef<HTMLLIElement, React.HTMLAttributes<HTMLLIElement>>(
  ({ className, ...props }, ref) => (
    <li ref={ref} className={cn("flex gap-3 items-end", className)} {...props}>
      <span className="text-sm text-accent-10" aria-label="More pages">
        <MoreHorizontal className="h-4 w-4 text-accent-10" />
      </span>
      <div className="text-accent-10" aria-hidden="true">
        /
      </div>
    </li>
  ),
);

Navbar.Breadcrumbs = Breadcrumbs;

Navbar.displayName = "GlobalNavbar";
Navbar.Actions.displayName = "GlobalNavbar.Actions";
Navbar.Breadcrumbs.displayName = "GlobalNavbar.Breadcrumbs";
Navbar.Breadcrumbs.Link.displayName = "GlobalNavbar.Breadcrumbs.Link";
Navbar.Breadcrumbs.Ellipsis.displayName = "GlobalNavbar.Breadcrumbs.Ellipsis";
