import { cn } from "@/lib/utils";

type QueriesPillType = {
  value: string | number;
};
export const QueriesPill = ({ value }: QueriesPillType) => {
  let color = undefined;
  let wording = value;
  if (value === 200 || value === "200") {
    color = "bg-success-9";
    wording = "2xx";
  } else if (value === 400 || value === "400") {
    color = "bg-warning-9";
    wording = "4xx";
  } else if (value === 500 || value === "500") {
    color = "bg-error-9";
    wording = "5xx";
  }
  return (
    <div className="h-6 bg-gray-3 inline-flex justify-start items-center py-1.5 px-2 rounded rounded-md gap-2">
      {color && <div className={cn("w-2 h-2 rounded-[2px]", color)} />}
      <span className="font-mono font-medium text-xs text-gray-12 text-xs ellipsis overflow-clip">
        {wording}
      </span>
    </div>
  );
};
