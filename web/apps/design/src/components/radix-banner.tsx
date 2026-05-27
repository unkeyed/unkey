interface Props {
  docsHref: string;
  apiHref: string;
}

export function RadixBanner({ docsHref, apiHref }: Props) {
  return (
    <div className="not-prose my-6 flex items-center justify-between rounded-xl border bg-muted/40 px-4 py-3">
      <div className="flex items-center gap-3">
        <svg
          aria-hidden="true"
          viewBox="0 0 25 25"
          className="size-4 text-foreground"
          fill="currentColor"
        >
          <path d="M12 25C5.925 25 1 20.075 1 14h11v11zM12 0v11H1C1 4.925 5.925 0 12 0zM18.5 11c3.038 0 5.5-2.462 5.5-5.5S21.538 0 18.5 0 13 2.462 13 5.5s2.462 5.5 5.5 5.5z" />
        </svg>
        <span className="font-medium text-foreground text-sm">This component uses Radix UI</span>
      </div>
      <div className="flex items-center gap-2">
        <a
          href={docsHref}
          target="_blank"
          rel="noreferrer"
          className="inline-flex h-7 items-center rounded-md border bg-background px-3 font-medium text-xs transition-colors hover:bg-accent"
        >
          Docs
        </a>
        <a
          href={apiHref}
          target="_blank"
          rel="noreferrer"
          className="inline-flex h-7 items-center rounded-md border bg-background px-3 font-medium text-xs transition-colors hover:bg-accent"
        >
          API Reference
        </a>
      </div>
    </div>
  );
}
