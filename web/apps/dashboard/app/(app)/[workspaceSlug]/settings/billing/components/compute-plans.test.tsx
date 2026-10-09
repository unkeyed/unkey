import type { DeployPlan } from "@/lib/stripe/deployPlan";
import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ComputePlans } from "./compute-plans";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: vi.fn() }),
  usePathname: () => "/acme/settings/billing",
}));

vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ slug: "acme" }),
}));

vi.mock("@/hooks/use-invalidate-workspace-queries", () => ({
  useInvalidateWorkspaceQueries: () => vi.fn(),
}));

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    useUtils: () => ({
      stripe: {
        getDeploySubscription: { invalidate: vi.fn() },
        getDeployEntitlement: { invalidate: vi.fn() },
        getUpcomingInvoice: { invalidate: vi.fn() },
        getDeployCredit: { invalidate: vi.fn() },
      },
      billing: { queryDeployUsage: { invalidate: vi.fn() } },
    }),
    stripe: {
      changeDeployPlan: {
        useMutation: () => ({ mutate: vi.fn(), isLoading: false }),
      },
      cancelDeploy: {
        useMutation: () => ({ mutate: vi.fn(), isLoading: false }),
      },
    },
  },
}));

afterEach(cleanup);

const option = (
  plan: DeployPlanOption["plan"],
  name: string,
  amount: number,
): DeployPlanOption => ({
  plan,
  name,
  amount,
  currency: "usd",
  interval: "month",
  description: null,
});

const CATALOG = [
  option("starter", "Starter", 500),
  option("pro", "Pro", 2500),
  option("business", "Business", 5000),
];

function renderPlans(
  signupCreditClaimed: boolean | undefined,
  options: DeployPlanOption[] = CATALOG,
  plans: readonly DeployPlan[] = ["starter", "pro", "business"],
) {
  render(
    <ComputePlans
      plans={plans}
      options={options}
      current={{ status: "none" }}
      usageCents={null}
      isAdmin
      from="billing"
      manage={false}
      onChanged={() => undefined}
      signupCreditClaimed={signupCreditClaimed}
    />,
  );
}

describe("Compute plan prices", () => {
  it("shows the regular price and the lowered first month for a new workspace", () => {
    renderPlans(false);

    expect(screen.getByText("$5", { exact: true }).className).toContain("line-through");
    expect(screen.getByText("first month $0")).toBeTruthy();
    expect(screen.getByText("$25", { exact: true }).className).toContain("line-through");
    expect(screen.getByText("first month $20")).toBeTruthy();
    expect(screen.getByText("$50", { exact: true }).className).toContain("line-through");
    expect(screen.getByText("first month $45")).toBeTruthy();
    expect(screen.getAllByText("$5 on Unkey")).toHaveLength(3);
  });

  it("keeps the regular price when the workspace already claimed the credit", () => {
    renderPlans(true);

    expect(screen.getByText("$25", { exact: true }).className).not.toContain("line-through");
    expect(screen.getByText("$50", { exact: true })).toBeTruthy();
    expect(screen.queryByText("first month $20")).toBeNull();
    expect(screen.queryByText("first month $45")).toBeNull();
    expect(screen.queryByText("first month $0")).toBeNull();
    expect(screen.queryByText("$5 on Unkey")).toBeNull();
  });

  it("hides the lowered price while the claim query is loading", () => {
    renderPlans(undefined);

    expect(screen.getByText("$25", { exact: true })).toBeTruthy();
    expect(screen.queryByText(/first month/)).toBeNull();
    expect(screen.queryByText("$5 on Unkey")).toBeNull();
  });

  it("floors the first month at $0", () => {
    renderPlans(
      false,
      [option("starter", "Starter", 400), option("pro", "Pro", 500)],
      ["starter", "pro"],
    );

    expect(screen.getByText("$4", { exact: true })).toBeTruthy();
    expect(screen.getByText("$5", { exact: true })).toBeTruthy();
    expect(screen.getAllByText("first month $0")).toHaveLength(2);
    expect(screen.queryByText(/-\$/)).toBeNull();
  });
});
