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
  updatePageNumber,
}: { currentPage: number; numPages: number; updatePageNumber: (pageNumber: number) => void }) {
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
          <PaginationLink
            isActive={currentPage === count ? true : false}
            onClick={() => updatePageNumber(count)}
          >
            {count}
          </PaginationLink>,
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
          <PaginationPrevious onClick={() => updatePageNumber(currentPage - 1)} />
        </PaginationItem>
        <GetPageButtons />
        <PaginationItem>
          <PaginationNext onClick={() => updatePageNumber(currentPage + 1)} />
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  );
}
