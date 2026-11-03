export type ListFooterState =
  | { kind: "hidden" }
  | { kind: "more"; onLoadMore: () => void }
  | { kind: "loadingMore" }
  | {
      kind: "error";
      reason: "refresh" | "loadMore";
      error: unknown;
      onRetry: () => void;
      retrying: boolean;
    };

export type FooterKind = ListFooterState["kind"];

type FooterErrorReason = Extract<ListFooterState, { kind: "error" }>["reason"];

export const FOOTER_ERROR_COPY: Record<FooterErrorReason, (plural: string) => string> = {
  refresh: (plural) => `We couldn't refresh ${plural}.`,
  loadMore: (plural) => `We couldn't load more ${plural}.`,
};
