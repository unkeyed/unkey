import { Navigation } from "@/components/navigation/navigation";
import { routes } from "@/lib/navigation/routes";
import { User } from "@unkey/icons";
import { Empty } from "@unkey/ui";

type Props = {
  params: Promise<{ workspaceSlug: string }>;
};

export default async function PortalPage({ params }: Props) {
  const { workspaceSlug } = await params;

  return (
    <div>
      <Navigation href={routes.portal.root({ workspaceSlug })} name="Portal" icon={<User />} />
      <Empty>
        <Empty.Title>Coming soon</Empty.Title>
        <Empty.Description>
          Portal configuration is on its way. Check back once it ships.
        </Empty.Description>
      </Empty>
    </div>
  );
}
