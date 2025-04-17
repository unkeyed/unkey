import { ApiListClient } from "./_components/api-list-client";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: { new?: boolean };
};

export default function ApisOverviewPage(props: Props) {
  return (
    <div>
      <Navigation isNewApi={!!props.searchParams.new} />
      <ApiListClient />
    </div>
  );
}
