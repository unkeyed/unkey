import { act, render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(),
}));

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    alerts: {
      series: {
        useQuery: mocks.useQuery,
      },
    },
  },
}));

vi.mock("@unkey/ui", () => ({
  InfoTooltip: "div",
  Skeleton: "div",
}));

import { LazyAlertRowChart } from "./alert-row-chart";

let observer: MockIntersectionObserver | undefined;

class MockIntersectionObserver implements IntersectionObserver {
  readonly root = null;
  readonly rootMargin = "150px 0px";
  readonly thresholds = [0];

  constructor(private readonly callback: IntersectionObserverCallback) {
    observer = this;
  }

  disconnect() {}
  observe() {}
  takeRecords(): IntersectionObserverEntry[] {
    return [];
  }
  unobserve() {}

  emit(isIntersecting: boolean) {
    const rect = new DOMRect();
    this.callback(
      [
        {
          boundingClientRect: rect,
          intersectionRatio: isIntersecting ? 1 : 0,
          intersectionRect: rect,
          isIntersecting,
          rootBounds: null,
          target: document.body,
          time: 0,
        },
      ],
      this,
    );
  }
}

const props = {
  appId: "app_1",
  environmentId: "env_1",
  metric: "requests" as const,
  windowStart: 1_000,
  windowEnd: 2_000,
};

describe("LazyAlertRowChart", () => {
  beforeEach(() => {
    observer = undefined;
    mocks.useQuery.mockReset();
    mocks.useQuery.mockReturnValue({ isLoading: true });
    vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
  });

  it("does not request a series while its row is off screen", () => {
    render(<LazyAlertRowChart {...props} />);

    expect(observer).toBeDefined();
    act(() => observer?.emit(false));

    expect(mocks.useQuery).not.toHaveBeenCalled();
  });

  it("requests a series when its row approaches the viewport", () => {
    render(<LazyAlertRowChart {...props} />);

    act(() => observer?.emit(true));

    expect(mocks.useQuery).toHaveBeenCalledOnce();
  });
});
