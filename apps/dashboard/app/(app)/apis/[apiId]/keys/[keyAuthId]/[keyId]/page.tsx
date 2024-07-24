import { ArrowLeft } from "lucide-react";
import Link from "next/link";

export default function APIKeyDetailPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  return (
    <div className="flex flex-col">
      <Link
        href={`/apis/${props.params.apiId}/keys/${props.params.keyAuthId}`}
        className="flex w-fit items-center gap-1 text-sm duration-200 text-content-subtle hover:text-secondary-foreground"
      >
        <ArrowLeft className="w-4 h-4" /> Back to API Keys listing
      </Link>

      {/* TODO: add table for Keys */}
    </div>
  );
}
