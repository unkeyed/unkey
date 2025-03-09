import { cn } from "@/lib/utils";

type QueriesPillType = {
  value: string | number;
  className?: string;
};
export const QueriesPill = ({ value, className }: QueriesPillType) => {
  const { color, phrase } = parseValue(value.toString());

  return (
    <div
      className={cn(
        "h-6 bg-gray-3 inline-flex justify-start items-center py-1.5 px-2  rounded-md gap-2 ",
        className,
      )}
    >
      {color && <div className={cn("w-2 h-2 rounded-[2px]", color)} />}
      <span className="font-mono text-xs font-medium truncate text-gray-12">{phrase}</span>
    </div>
  );
};

const parseValue = (value: string) => {
  // Check if value can be parsed as a number

  const isNumeric = !Number.isNaN(Number.parseFloat(value)) && Number.isFinite(Number(value));
  if (!isNumeric) {
    return { color: null, phrase: value };
  }
  const numValue = Number(value);
  if (numValue >= 200 && numValue < 300) {
    return { color: "bg-success-9", phrase: value === "200" ? "2xx" : value };
  }
  if (numValue >= 400 && numValue < 500) {
    return { color: "bg-warning-9", phrase: value === "400" ? "4xx" : value };
  }
  if (numValue >= 500 && numValue < 600) {
    return { color: "bg-error-9", phrase: value === "500" ? "5xx" : value };
  }
  return { color: null, phrase: value };
};
