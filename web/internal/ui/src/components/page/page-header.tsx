import { mergeProps } from "@base-ui/react/merge-props";
import { useRender } from "@base-ui/react/use-render";
import { IconChevronLeftOutline12 } from "@unkey/icons";
import type * as React from "react";
import { cn } from "../../lib/utils";

function PageHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-3 px-6 pt-6 2xl:pt-8",
        "has-[[data-slot=page-header-description]]:items-start",
        "group-data-[width=full]/page:max-w-none group-data-[width=full]/page:border-b group-data-[width=full]/page:pb-4 group-data-[width=full]/page:pt-4",
        className,
      )}
      {...props}
    />
  );
}

function PageHeaderContent({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex flex-col gap-0.5 min-w-0", className)} {...props} />;
}

type PageHeaderBackProps = useRender.ComponentProps<"a">;

function PageHeaderBack({ className, render, children, ...props }: PageHeaderBackProps) {
  return useRender({
    defaultTagName: "a",
    render,
    props: mergeProps<"a">(
      {
        className: cn(
          "-ml-1 flex w-fit items-center gap-1 rounded-md px-1 py-0.5 text-sm text-gray-10 transition-colors hover:text-gray-12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7",
          className,
        ),
        children: (
          <>
            <IconChevronLeftOutline12 />
            {children}
          </>
        ),
      },
      props,
    ),
  });
}

function PageHeaderTitle({ className, ...props }: React.ComponentProps<"h1">) {
  return (
    <h1
      className={cn(
        "text-xl font-semibold tracking-tight leading-tight text-gray-12 m-0",
        className,
      )}
      {...props}
    />
  );
}

function PageHeaderDescription({ className, ...props }: React.ComponentProps<"p">) {
  return (
    <p
      data-slot="page-header-description"
      className={cn("text-sm leading-5 text-gray-11 m-0", className)}
      {...props}
    />
  );
}

function PageHeaderActions({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex flex-wrap items-center gap-2 shrink-0", className)} {...props} />;
}

export {
  PageHeader,
  PageHeaderContent,
  PageHeaderBack,
  PageHeaderTitle,
  PageHeaderDescription,
  PageHeaderActions,
};
