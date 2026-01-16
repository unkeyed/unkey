import { LLMSearch, toast } from "@unkey/ui";

export const LogsSearch = ({ identityId: _identityId }: { identityId: string }) => {
  // For now, we'll create a placeholder search that shows a message
  // In the future, this could be enhanced with identity-specific LLM search
  const handleSearch = async (_query: string) => {
    toast.info(
      "Identity-specific search is coming soon. Use the filters below to narrow down results.",
      {
        duration: 5000,
        position: "top-right",
      },
    );
  };

  return (
    <LLMSearch
      exampleQueries={["Show rate limited outcomes", "Filter by successful verifications"]}
      isLoading={false}
      searchMode="manual"
      onSearch={handleSearch}
    />
  );
};
