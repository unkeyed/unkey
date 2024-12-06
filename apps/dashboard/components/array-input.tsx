import { Badge } from "@/components/ui/badge";
import { CornerDownLeft, X } from "lucide-react";
import { useCallback, useState } from "react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

type Props = {
  title?: string;
  placeholder?: string;
  selected: string[];
  setSelected: (v: string[]) => void;
};

export const ArrayInput: React.FC<Props> = ({ title, placeholder, selected, setSelected }) => {
  console.log(selected);
  const [items, setItems] = useState<string[]>(selected);
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
    [inputValue, items],
  );

  return (
    <div className="bg-transparent flex flex-col relative">
      <div>
        {title && <span className="text-xs font-medium absolute -top-5">{title}:</span>}
        <div className="flex gap-1 absolute left-[68px] -top-[26px]">
          {items?.map((item) => (
            <Badge key={item} variant="secondary">
              {item}
              <button
                type="button"
                className="ml-1 rounded-full outline-none"
                onClick={() => handleUnselect(item)}
              >
                <X className="w-3 h-3 text-content-muted hover:text-content" />
              </button>
            </Badge>
          ))}
        </div>
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
        <Button size="icon" variant="secondary" onClick={handleAdd}>
          <CornerDownLeft className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
};
