import { Plus } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

type EnvVarsEmptyProps = {
  searchQuery: string;
};

export function EnvVarsEmpty({ searchQuery }: EnvVarsEmptyProps) {
  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      <div className="flex items-center justify-center py-16 px-4">
        <Empty className="w-[400px] flex items-start">
          <Empty.Icon className="w-auto" />
          <Empty.Title>
            {searchQuery ? "No Matching Variables" : "No Environment Variables"}
          </Empty.Title>
          <Empty.Description className="text-left">
            {searchQuery
              ? `No variables matching "${searchQuery}". Try a different search term.`
              : "Environment variables will appear here once you add them. Store API keys, tokens, and config securely."}
          </Empty.Description>
          {!searchQuery && (
            <Empty.Actions className="mt-4 justify-start">
              <Button
                size="md"
                onClick={() => {
                  // TODO: wire up add action
                }}
              >
                <Plus iconSize="sm-regular" />
                Add Environment Variable
              </Button>
            </Empty.Actions>
          )}
        </Empty>
      </div>
    </div>
  );
}
