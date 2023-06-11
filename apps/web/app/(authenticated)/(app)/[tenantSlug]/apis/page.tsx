import { PageHeader } from "@/components/PageHeader";
import { CreateApiButton } from "./CreateAPI";
import { getTenantId } from "@/lib/auth";
export default function OverviewPage() {
  const tenantId = getTenantId();
  return (
    <div>
      <PageHeader
        title="Applications"
        description="Manage your different APIs"
        actions={[<CreateApiButton key="createApi" tenantId={tenantId} />]}
      />
    </div>
  );
}
