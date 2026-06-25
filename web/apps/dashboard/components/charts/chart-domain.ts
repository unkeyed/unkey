// When the caller opts into contraction and non-zero data covers less than
// this fraction of the requested window, the x-axis contracts to the data's
// extent. 7 days of axis with 20 minutes of data reads as broken; showing the
// 20 minutes and letting the axis labels carry the real span is clearer. 0.15
// = 15% is the gut-check minimum for "full enough to keep the window".
export const SPARSE_COVERAGE_THRESHOLD = 0.15;

// A degenerate non-zero span (a single bucket, or no non-zero data at all)
// would contract to a zero-width domain and the lone point would vanish.
// Below this we keep the requested window so the point has a frame to sit in.
const DEGENERATE_SPAN_MS = 60_000;

export type XAxisDomainResult = {
  // [start, end] the x-axis should span, or undefined to let the chart infer
  // it from the data extent (used when no window was requested).
  effectiveDomain: [number, number] | undefined;
  // Width of the rendered span in millis. Drives tick formatting: a sub-2-day
  // span reads as times, a wider one as dates.
  spanMs: number;
};

// resolveXAxisDomain decides whether the chart honors the caller's requested
// window or contracts to the data's non-zero extent.
//
// It anchors to the window by default. Only when `contractOnSparseData` is set
// AND the data is genuinely sparse (coverage below the threshold, but not a
// degenerate single-point span) does it contract. Honoring the window by
// default keeps a header that commits to it (e.g. "requests this week") in
// agreement with the axis, which is why the overview card leaves contraction
// off while the deployment resource panels turn it on.
//
// `firstNonZeroTs`/`lastNonZeroTs` are the bounds of the non-zero samples,
// not the raw data extent: ClickHouse's WITH FILL pads every bucket in the
// window, so the raw extent would always report 100% coverage.
export function resolveXAxisDomain(params: {
  xAxisDomain?: [number, number];
  contractOnSparseData?: boolean;
  firstNonZeroTs?: number;
  lastNonZeroTs?: number;
}): XAxisDomainResult {
  const { xAxisDomain, contractOnSparseData, firstNonZeroTs, lastNonZeroTs } = params;

  const nonZeroSpanMs =
    firstNonZeroTs !== undefined && lastNonZeroTs !== undefined
      ? Math.max(0, lastNonZeroTs - firstNonZeroTs)
      : 0;
  const windowSpanMs = xAxisDomain ? xAxisDomain[1] - xAxisDomain[0] : 0;
  const coverage = windowSpanMs > 0 ? nonZeroSpanMs / windowSpanMs : 0;

  const useAnchoredDomain =
    Boolean(xAxisDomain) &&
    (!contractOnSparseData ||
      coverage >= SPARSE_COVERAGE_THRESHOLD ||
      nonZeroSpanMs < DEGENERATE_SPAN_MS);

  if (useAnchoredDomain && xAxisDomain) {
    return { effectiveDomain: xAxisDomain, spanMs: windowSpanMs };
  }

  // Contract to the non-zero bounds explicitly rather than letting the chart
  // use the full data extent, which would re-include the zero-filled padding
  // and collapse the sparse data into a right-hand sliver again.
  const contracted: [number, number] | undefined =
    firstNonZeroTs !== undefined && lastNonZeroTs !== undefined
      ? [firstNonZeroTs, lastNonZeroTs]
      : undefined;
  return { effectiveDomain: contracted, spanMs: nonZeroSpanMs };
}
