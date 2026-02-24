import { Avatar, AvatarImage } from "@/components/ui/avatar";
import { CircleHalfDottedClock } from "@unkey/icons";

type QueriesMadeByProps = {
  userName?: string;
  userImageSrc: string;
  createdString: string;
};

export const QueriesMadeBy = ({ userName, userImageSrc, createdString }: QueriesMadeByProps) => {
  return (
    <div className="flex flex-row items-center justify-start w-full h-6 gap-2 mt-2">
      {/* User Avatar */}
      {userName && (
        <>
          <span className="font-mono text-xs font-normal text-gray-9">by</span>
          <Avatar className="h-[21px] w-[21px]">
            <AvatarImage
              src={userImageSrc}
              alt={userName}
              className="rounded-full border border-gray-4"
            />
          </Avatar>
          <span className="font-mono text-xs font-medium leading-4 text-gray-12">{userName}</span>
        </>
      )}
      <CircleHalfDottedClock className="size-3.5 text-gray-12 mb-[2px] ml-[2px]" />
      <span className="font-mono text-xs font-normal leading-4 text-gray-9">{createdString}</span>
    </div>
  );
};
