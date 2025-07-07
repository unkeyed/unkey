// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../lib/utils";

type InlineLinkProps = {
  /**
   * The URL to link to.
   */
  href: string;
  /**
   * The label to display.
   */
  label?: string;
  /**
   * The decorative icon to display alongside the link text.
   * Should be purely decorative and not convey meaning.
   */
  icon?: React.ReactNode;
  /**
   * The position of the decorative icon relative to the text.
   */
  iconPosition?: "left" | "right";
  /**
   * Additional CSS classes to apply to the link.
   */
  className?: string;
  /**
   * The target attribute for the link. Standard HTML values: "_blank", "_self", "_parent", "_top", or a frame name.
   */
  target?: "_blank" | "_self" | "_parent" | "_top";
  
};
const InlineLink: React.FC<InlineLinkProps> = ({
  className,
  label,
  href,
  icon,
  iconPosition = "right",
  target,
  ...props
}) => {
  return (
    <a
      href={href}
      className={cn(className, "underline inline-flex hover:opacity-70")}
      target={target}
      rel={target === "_blank" ? "noopener noreferrer" : undefined}
      {...props}
    >
      <span className="inline-flex gap-x-1 items-center">
        {iconPosition === "left" && icon && <span aria-hidden="true">{icon}</span>}
        {label}
        {iconPosition === "right" && icon && <span aria-hidden="true">{icon}</span>}
      </span>
    </a>
  );
};

InlineLink.displayName = "InlineLink";

export { InlineLink };
