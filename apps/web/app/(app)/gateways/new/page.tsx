import { PageHeader } from "@/components/dashboard/page-header";
import { Shuffle } from "lucide-react";
import { CreateGatewayForm } from "./form";

export const dynamic = "force-dynamic";
export default async function CreateGatewayPage() {
  return (
    <div>
      <PageHeader title="Create your new Gateway" description="Proxy your OpenAI key securely" />

      <div className="flex items-start justify-between gap-16 mt-8">
        <div className="space-y-2">
          <div className="inline-flex items-center justify-center p-4 border rounded-full bg-primary/5">
            <Shuffle className="w-6 h-6 text-primary" />
          </div>
          <h4 className="text-lg font-medium">What are Gateways?</h4>
          <p className="text-sm text-content-subtle">Gateways allow you to rewrite HTTP headers.</p>
          <ol className="ml-2 space-y-1 text-sm list-decimal list-outside text-content-subtle">
            <li>A user calls your gateway</li>
            <li>The request gets validated</li>
            <li>Unkey adds headers as configured and sends the request to the origin</li>
            <li>Unkey returns the response from the origin back to your user</li>
          </ol>
        </div>

        <div className="w-full">
          <CreateGatewayForm />
        </div>
      </div>
    </div>
  );
}
