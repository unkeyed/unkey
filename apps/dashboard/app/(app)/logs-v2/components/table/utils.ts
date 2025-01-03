function generateMockLog(overrides = {}) {
  // Helper function to generate random string
  const generateRandomString = (length = 10) => {
    return Math.random()
      .toString(36)
      .substring(2, length + 2);
  };

  // Helper function to generate random HTTP method
  const generateRandomMethod = () => {
    const methods = ["GET", "POST", "PUT", "DELETE", "PATCH"];
    return methods[Math.floor(Math.random() * methods.length)];
  };

  // Helper function to generate random path
  const generateRandomPath = () => {
    const paths = [
      "/api/v1/users",
      "/api/v1/workspaces",
      "/api/v1/logs",
      "/api/v1/settings",
      "/health",
      "/metrics",
    ];
    return paths[Math.floor(Math.random() * paths.length)];
  };

  // Helper function to generate random headers
  const generateRandomHeaders = () => {
    const possibleHeaders = [
      "Content-Type: application/json",
      "Authorization: Bearer token123",
      "X-Request-ID: req123",
      "Accept: application/json",
      "User-Agent: Mozilla/5.0",
    ];
    const numHeaders = Math.floor(Math.random() * 3) + 1; // 1-3 headers
    return possibleHeaders.slice(0, numHeaders);
  };

  // Generate base mock log
  const mockLog = {
    request_id: `req_${generateRandomString(8)}`,
    time: Math.floor(Date.now() / 1000),
    workspace_id: `ws_${generateRandomString(8)}`,
    host: `${generateRandomString(5)}.example.com`,
    method: generateRandomMethod(),
    path: generateRandomPath(),
    request_headers: generateRandomHeaders(),
    request_body: JSON.stringify({ data: generateRandomString() }),
    response_status:
      Math.random() < 0.6 ? 200 : Math.random() < 0.5 ? 500 : 400, // 80% success rate
    response_headers: generateRandomHeaders(),
    response_body: JSON.stringify({
      keyId: "key_2Krf19pCiGx5UE29qJeBu7JpTzHk",
      valid: true,
      meta: {
        hello: "world",
      },
      enabled: true,
    }),
    error: Math.random() < 0.8 ? "" : "Internal Server Error",
    service_latency: Math.floor(Math.random() * 1000), // 0-1000ms
  };

  // Apply any overrides
  return {
    ...mockLog,
    ...overrides,
  };
}

// Generate multiple logs
export const generateMockLogs = (count: number, overrides = {}) => {
  return Array.from({ length: count }, () => generateMockLog(overrides));
};
