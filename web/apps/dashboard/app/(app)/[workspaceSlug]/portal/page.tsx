import { routes } from "@/lib/navigation/routes";
import {
  Button,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";

type Props = {
  params: Promise<{ workspaceSlug: string }>;
};

// A portal is configured per API, on that API's Customer portal page. There is
// no workspace-level list because the API exposes no way to enumerate a
// workspace's portals -- only lookup by portal id, slug, or mapped resource.
export default async function PortalPage({ params }: Props) {
  const { workspaceSlug } = await params;

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Portal</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <Empty>
          <Empty.Title>Portals are set up per API</Empty.Title>
          <Empty.Description>
            Each customer portal serves the keys of a single API. Open an API and choose
            &ldquo;Customer portal&rdquo; to create or configure one.
          </Empty.Description>
          <Empty.Actions>
            <Button variant="primary" render={<Link href={routes.apis.list({ workspaceSlug })} />}>
              Go to APIs
            </Button>
          </Empty.Actions>
        </Empty>
      </PageBody>
    </PageContainer>
  );
}
