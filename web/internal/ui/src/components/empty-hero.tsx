import React from "react";
import { cn } from "../lib/utils";

type EmptyHeroProps = React.HTMLAttributes<HTMLDivElement>;

/**
 * A full-width bordered empty state for page bodies: an {@link EmptyHero.Icons}
 * row, title, description, and actions, centered inside a card-like shell.
 * Type scale and spacing are owned by the sub-components so consumers stay
 * visually consistent.
 */
function EmptyHero({ className, children, ...props }: EmptyHeroProps) {
  return (
    <div
      className={cn(
        "flex w-full flex-col items-center justify-center text-center border border-grayA-4 rounded-lg px-8 py-24",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
}
EmptyHero.displayName = "EmptyHero";

const OPACITY_BY_DISTANCE_FROM_CENTER = ["opacity-90", "opacity-75", "opacity-50"];

type EmptyHeroIconsProps = Omit<React.HTMLAttributes<HTMLDivElement>, "style">;

/**
 * A decorative row of icons that fades out at both edges. Pass 3-5 icons; the
 * center one sits in a larger box and should use a thinner stroke (e.g.
 * `iconSize="md-thin"` vs `md-medium` for the flanks) — the boxes control the
 * rendered icon size.
 */
EmptyHero.Icons = function EmptyHeroIcons({ className, children, ...props }: EmptyHeroIconsProps) {
  const icons = React.Children.toArray(children);
  const centerIndex = Math.floor(icons.length / 2);
  return (
    <div
      aria-hidden="true"
      className={cn("p-2 mb-5", className)}
      style={{
        maskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
        WebkitMaskImage:
          "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
      }}
      {...props}
    >
      <div className="flex gap-6 items-center justify-center text-gray-12">
        {icons.map((icon, index) => {
          const distanceFromCenter = Math.min(
            Math.abs(index - centerIndex),
            OPACITY_BY_DISTANCE_FROM_CENTER.length - 1,
          );
          return (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: static decorative row, index is stable
              key={index}
              className={cn(
                "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none",
                index === centerIndex ? "size-16 [&_svg]:size-9" : "size-9 [&_svg]:size-[18px]",
                OPACITY_BY_DISTANCE_FROM_CENTER[distanceFromCenter],
              )}
            >
              {icon}
            </div>
          );
        })}
      </div>
    </div>
  );
};

type EmptyHeroTitleProps = React.HTMLAttributes<HTMLHeadingElement>;

EmptyHero.Title = function EmptyHeroTitle({ className, ...props }: EmptyHeroTitleProps) {
  return (
    <h2
      className={cn("text-accent-12 mt-3 font-semibold text-[15px] leading-6", className)}
      {...props}
    />
  );
};

type EmptyHeroDescriptionProps = React.HTMLAttributes<HTMLParagraphElement>;

EmptyHero.Description = function EmptyHeroDescription({
  className,
  ...props
}: EmptyHeroDescriptionProps) {
  return (
    <p
      className={cn(
        "text-accent-11 text-center text-sm font-normal leading-6 mt-1 max-w-md text-balance",
        className,
      )}
      {...props}
    />
  );
};

type EmptyHeroActionsProps = React.HTMLAttributes<HTMLDivElement>;

EmptyHero.Actions = function EmptyHeroActions({
  className,
  children,
  ...props
}: EmptyHeroActionsProps) {
  return (
    <div className={cn("w-full flex items-center justify-center gap-3 mt-6", className)} {...props}>
      {children}
    </div>
  );
};

export { EmptyHero };
