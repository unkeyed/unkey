import {
  differenceInDays,
  differenceInHours,
  differenceInMinutes,
  differenceInMonths,
  differenceInSeconds,
  differenceInWeeks,
  differenceInYears,
} from "date-fns";
import {
  Bookmark,
  ChartActivity2,
  CircleCheck,
  Clock,
  Conversion,
  Layers2,
  Link4,
} from "@unkey/icons";



export const getSinceTime = (date: number) => {
  const now = new Date();
  const seconds = differenceInSeconds(now, date);
  if (seconds < 60) {
    return "just now";
  }
  const minutes = differenceInMinutes(now, date);
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = differenceInHours(now, date);
  if (hours < 24) {
    return `${hours}h ago`;
  }
  const days = differenceInDays(now, date);
  if (days < 7) {
    return `${days}d ago`;
  }

  const weeks = differenceInWeeks(now, date);
  if (weeks < 4) {
    return `${weeks}w ago`;
  }

  const months = differenceInMonths(now, date);
  if (months < 12) {
    return `${months} month(s) ago`;
  }

  const years = differenceInYears(now, date);
  return `${years} year(s) ago`;
};


export const handleQueryKeyboard = (
  e: React.KeyboardEvent,
  containerRef: React.RefObject<HTMLElement>,
  focusedTabIndex: number,
  setFocusedTabIndex: React.Dispatch<React.SetStateAction<number>>,
  selectedQueryIndex: number,
  setSelectedQueryIndex: React.Dispatch<React.SetStateAction<number>>,
  filterGroups: any[],
  savedGroups: any[],
  handleSelectedQuery: (index: number) => void,
  setOpen: React.Dispatch<React.SetStateAction<boolean>>
) => {
    // Adjust scroll speed as needed

    if (containerRef.current) {
      const scrollSpeed = 50;
      // Handle up/down navigation
      if (e.key === "ArrowUp" || e.key === "k" || e.key === "K") {
        e.preventDefault();

        const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
        const totalItems = currentList.length - 1;
        containerRef.current.scrollTop -= scrollSpeed;
        if (totalItems === 0) {
          return;
        }

        // Move selection up, wrap to bottom if at top
        setSelectedQueryIndex((prevIndex) => (prevIndex > 0 ? prevIndex - 1 : 0));
      } else if (e.key === "ArrowDown" || e.key === "j" || e.key === "J") {
        e.preventDefault();

        containerRef.current.scrollTop += scrollSpeed;
        const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
        const totalItems = currentList.length - 1;

        if (totalItems === 0) {
          return;
        }

        // Move selection down, wrap to top if at bottom
        setSelectedQueryIndex((prevIndex) =>
          prevIndex < totalItems - 1 ? prevIndex + 1 : totalItems,
        );
      }
    }
    // Handle tab navigation
    if (e.key === "ArrowLeft" || e.key === "h" || e.key === "H") {
      // Move to All tab

          // Adjust scroll speed as needed
          
          // Rest of the function remains the same
          setFocusedTabIndex(0);
      setSelectedQueryIndex(0);
    } else if (e.key === "ArrowRight" || e.key === "l" || e.key === "L") {
      // Move to Saved tab
      setFocusedTabIndex(1);
      setSelectedQueryIndex(0);
    } else if (e.key === "Enter" || e.key === " ") {
      // Apply the selected filter
      const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
      if (currentList.length > 0 && selectedQueryIndex < currentList.length) {
        handleSelectedQuery(selectedQueryIndex);
        setOpen(false);
      }
    }
  };

export const getIcon = ({ field }: { field: string }) => {
  // Get the appropriate icon based on the field name
  switch (field.toLowerCase()) {
    case "status":
      return ChartActivity2;
    case "time":
    case "since":
    case "date":
      return Clock;
    case "tag":
      return Bookmark ;
    case "user":
      return Link4;
    case "layer":
    case "layers":
      return Layers2;
    case "success":
    case "verified":
      return CircleCheck ;
    case "conversion":
    case "convert":
      return Conversion;
    default:
      return ChartActivity2; // Default icon
  }
};
 

