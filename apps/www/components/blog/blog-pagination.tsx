import Link from "next/link";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "../ui/pagination";

export function BlogPagination({
  currentPage,
  numPages,
  buildPath,
}: { currentPage: number; numPages: number; buildPath: (page: number) => string }) {
  if (numPages === 1) {
    return null;
  }

  function GetPageButtons() {
    const content = [];
    for (let count = 1; count <= numPages; count++) {
      const isEllipses =
        (count > currentPage + 2 && count === numPages) ||
        (count <= currentPage - 2 && count === 2);

      if (!isEllipses) {
        content.push(
          <Link prefetch href={buildPath(count)}>
            <PaginationLink isActive={currentPage === count ? true : false}>{count}</PaginationLink>
          </Link>,
        );
      } else {
        content.push(<PaginationEllipsis />);
      }
    }
    return content;
  }

  return (
    <Pagination>
      <PaginationContent>
        <PaginationItem>
          <Link prefetch href={buildPath(currentPage - 1)}>
            <PaginationPrevious />
          </Link>
        </PaginationItem>
        <GetPageButtons />
        <PaginationItem>
          <Link prefetch href={buildPath(currentPage + 1)}>
            <PaginationNext />
          </Link>
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  );
}
