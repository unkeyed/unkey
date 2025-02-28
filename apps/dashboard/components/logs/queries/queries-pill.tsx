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
  const firstChar = value.toString().charAt(0);

  const parsedValues =
    firstChar === "2"
      ? { color: "bg-success-9", phrase: value !== "200" ? value : "2xx" }
      : firstChar === "4"
        ? { color: "bg-warning-9", phrase: value !== "400" ? value : "4xx" }
        : firstChar === "5"
          ? { color: "bg-error-9", phrase: value !== "500" ? value : "5xx" }
          : { color: null, phrase: value };
  return parsedValues;
};
