import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
} from "@/components/ui/breadcrumb";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default function PageBreadcrumb() {
  return (
    <div className="flex items-center justify-between">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink href="/identities">Identities</BreadcrumbLink>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="bg-background border text-content-subtle rounded text-xs px-1 py-0.5  font-mono ">
        Beta
      </div>
    </div>
  );
}
