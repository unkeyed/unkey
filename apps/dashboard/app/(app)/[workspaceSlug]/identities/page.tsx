"use client";
import { OptIn } from "@/components/opt-in";
import { PageContent } from "@/components/page-content";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Empty, Loading } from "@unkey/ui";
import { Suspense } from "react";
import { z } from "zod";
import { Results } from "./components/results";
import { SearchField } from "./filter";
import { Navigation } from "./navigation";
export const dynamic = "force-dynamic";

type Props = {
  searchParams: {
    search?: string;
    limit?: string;
  };
};

const DEFAULT_LIMIT = 100;

const searchParamsSchema = z.object({
  search: z.string().optional(),
  limit: z.string().regex(/^\d+$/).optional(),
});

export default function Page(props: Props) {
  const validatedParams = searchParamsSchema.parse(props.searchParams);
  const search = validatedParams.search ?? "";
  const limit = validatedParams.limit ? Number.parseInt(validatedParams.limit, 10) : DEFAULT_LIMIT;

  const workspace = useWorkspaceNavigation();

  if (!workspace.betaFeatures.identities) {
    return <OptIn title="Identities" description="Identities are in beta" feature="identities" />;
  }

  return (
    <div>
      <Navigation />
      <PageContent>
        <SearchField />
        <div className="flex flex-col gap-8 mb-20 mt-8">
          <Suspense
            fallback={
              <Empty>
                <Empty.Title>
                  <Loading />
                </Empty.Title>
              </Empty>
            }
          >
            <Results search={search ?? ""} limit={limit ?? DEFAULT_LIMIT} />
          </Suspense>
        </div>
      </PageContent>
    </div>
  );
}
