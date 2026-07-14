import {
  Button,
  Input,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  ResourceList,
  ResourceListContent,
  ResourceListFooter,
  ResourceListHeader,
} from "@unkey/ui";
import { useState } from "react";

const initialIdentities = [
  { externalId: "customer_123", rateLimits: 2 },
  { externalId: "customer_456", rateLimits: 1 },
  { externalId: "customer_789", rateLimits: 4 },
];

export function ResourceListPageExample() {
  const [identities, setIdentities] = useState(initialIdentities);
  const [query, setQuery] = useState("");
  const [visibleCount, setVisibleCount] = useState(2);
  const filteredIdentities = identities.filter((identity) =>
    identity.externalId.toLowerCase().includes(query.trim().toLowerCase()),
  );
  const visibleIdentities = filteredIdentities.slice(0, visibleCount);

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <h2 className="m-0 font-semibold text-[22px] text-accent-12 leading-tight tracking-tight">
            Identities
          </h2>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            variant="outline"
            onClick={() =>
              setIdentities((current) => [
                { externalId: `customer_${1000 + current.length}`, rateLimits: 0 },
                ...current,
              ])
            }
          >
            Create identity
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <ResourceList>
          <ResourceListHeader>
            <Input
              aria-label="Search identities"
              placeholder="Search identities"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </ResourceListHeader>
          <ResourceListContent>
            <ul className="divide-y divide-grayA-4">
              {visibleIdentities.map((identity) => (
                <li
                  key={identity.externalId}
                  className="relative flex items-center justify-between px-4 py-3"
                >
                  <span className="font-mono text-accent-12 text-sm">{identity.externalId}</span>
                  <span className="text-gray-10 text-xs">{identity.rateLimits} rate limits</span>
                </li>
              ))}
              {visibleIdentities.length === 0 ? (
                <li className="px-4 py-8 text-center text-gray-10 text-sm">No identities found</li>
              ) : null}
            </ul>
            {visibleIdentities.length < filteredIdentities.length ? (
              <ResourceListFooter>
                <Button
                  variant="outline"
                  onClick={() => setVisibleCount(filteredIdentities.length)}
                >
                  Load more
                </Button>
              </ResourceListFooter>
            ) : null}
          </ResourceListContent>
        </ResourceList>
      </PageBody>
    </PageContainer>
  );
}
