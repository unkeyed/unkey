import { PaginationControls } from "@unkey/ui";
import { useState } from "react";

const TOTAL = 50;
const PAGE_SIZE = 15;
const LAST_PAGE = Math.ceil(TOTAL / PAGE_SIZE) - 1;

export default function PaginationBasic() {
  const [page, setPage] = useState(0);
  const start = page * PAGE_SIZE;
  const end = Math.min(start + PAGE_SIZE, TOTAL);

  return (
    <div className="mx-auto flex h-10 w-full max-w-sm items-center justify-between rounded-lg border pr-2 pl-4 text-grayA-9 text-xs">
      <span className="tabular-nums">
        {start + 1}–{end} of {TOTAL}
      </span>
      <PaginationControls
        label="Variables pagination"
        hasPrevious={page > 0}
        hasNext={page < LAST_PAGE}
        onPrevious={() => setPage(page - 1)}
        onNext={() => setPage(page + 1)}
      />
    </div>
  );
}
