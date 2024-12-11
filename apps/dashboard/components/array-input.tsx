import { Badge } from "@/components/ui/badge";
import { Button } from "@unkey/ui";
import { CornerDownLeft, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Input } from "./ui/input";

type Props = {
  title?: string;
  placeholder?: string;
  selected: string[];
  setSelected: (v: string[]) => void;
};

export const ArrayInput: React.FC<Props> = ({ title, placeholder, selected, setSelected }) => {
  const [items, setItems] = useState<string[]>(selected);

  useEffect(() => {
    setItems(selected);
  }, [selected]);
  const [inputValue, setInputValue] = useState("");

  const handleUnselect = (item: string) => {
    const newItems = items.filter((i) => i !== item);
    setItems(newItems);
    setSelected(newItems);
  };

  const handleAdd = () => {
    if (inputValue.trim()) {
      const newItems = Array.from(new Set([...items, inputValue.trim()]));
      setItems(newItems);
      setSelected(newItems);
      setInputValue("");
    }
  };

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        e.preventDefault();
        handleAdd();
      }
    },
    [handleAdd],
  );

  return (
    <div className="bg-transparent flex-col relative max-w-96">
      <div className="flex flex-row gap-2">
        <div className="flex flex-col justify-end">
          {title && <span className="text-xs font-medium mb-2 ">{title}:</span>}
        </div>
        <ul className="flex flex-wrap gap-1 mb-2 list-none p-0" aria-label="Selected items">
          {items?.map((item) => (
            <li key={item}>
              <Badge variant="secondary">
                {item}
                <button
                  type="button"
                  className="ml-1 rounded-full outline-none"
                  onClick={() => handleUnselect(item)}
                  aria-label={`Remove ${item}`}
                >
                  <X className="w-3 h-3 text-content-muted hover:text-content" />
                </button>
              </Badge>
            </li>
          ))}
        </ul>
      </div>
      <div className="flex items-center justify-center gap-2">
        <div className="flex flex-wrap items-center w-full gap-1">
          <Input
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            className="flex-1 w-full bg-transparent outline-none placeholder:text-content-subtle"
          />
        </div>
        <Button shape="square" variant="default" onClick={handleAdd}>
          <CornerDownLeft />
        </Button>
      </div>
    </div>
  );
};
