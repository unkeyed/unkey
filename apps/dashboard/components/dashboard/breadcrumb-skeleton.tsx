import { Breadcrumb, BreadcrumbList, BreadcrumbSeparator } from "@/components/ui/breadcrumb";
import { Skeleton } from "../ui/skeleton";

export function BreadcrumbSkeleton(props: { levels: number }) {
  return (
    <Breadcrumb>
      <BreadcrumbList>
        {Array.from({ length: props.levels }).map((_, idx) => (
          <>
            <Skeleton className="w-10 h-4 rounded-md" />
            {idx < props.levels - 1 && <BreadcrumbSeparator />}
          </>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
