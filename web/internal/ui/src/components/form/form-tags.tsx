import { cn } from "../../lib/utils";

type TagProps = {
  className?: string;
};

type DocsTagProps = TagProps & {
  href: string;
};

type RequiredTagProps = TagProps & {
  hasError?: boolean;
};

export const OptionalTag = ({ className }: TagProps) => {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-sm border border-grayA-4 text-grayA-11 px-1 py-0.5 text-xs font-sans bg-grayA-3 ml-2",
        className,
      )}
    >
      Optional
    </span>
  );
};

export const DocsTag = ({ className, href }: DocsTagProps) => {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        "inline-flex items-center rounded-sm border border-accent-4 text-accent-11 px-1 py-0.5 text-xs font-sans bg-accent-3 ml-2 no-underline hover:bg-accent-4 transition-colors",
        className,
      )}
      onClick={(e) => e.stopPropagation()}
    >
      Docs
    </a>
  );
};

export const RequiredTag = ({ className, hasError }: RequiredTagProps) => {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-sm border px-1 py-0.5 text-xs font-sans ml-2",
        hasError
          ? "border-error-4 text-error-11 bg-error-3"
          : "border-warning-4 text-warning-11 bg-warning-3 dark:border-warning-4 dark:text-warning-11 dark:bg-warning-3",
        className,
      )}
    >
      Required
    </span>
  );
};
