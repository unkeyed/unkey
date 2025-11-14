import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

type EmptyTableProps = {
  title: string;
  description: string | null;
  icon?: React.ReactNode;
};

export const EmptyTable = ({ title, description, icon }: EmptyTableProps) => {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-[400px] flex items-start">
        <Empty.Icon className="w-auto">{icon}</Empty.Icon>
        <Empty.Title>{title}</Empty.Title>
        <Empty.Description className="text-left">{description}</Empty.Description>
        <Empty.Actions className="mt-4 justify-center md:justify-start">
          <a
            href="https://www.unkey.com/docs/introduction"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button size="md">
              <BookBookmark />
              Documentation
            </Button>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
};
