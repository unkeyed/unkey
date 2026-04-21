type ErrorPageProps = {
  title: string;
  message: string;
};

function ErrorLayout({ title, message }: ErrorPageProps) {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center max-w-md px-4">
        <h1 className="text-2xl font-semibold text-gray-12">{title}</h1>
        <p className="mt-2 text-gray-11">{message}</p>
      </div>
    </div>
  );
}

/** No `?session` query parameter provided. */
export function NoSessionError() {
  return (
    <ErrorLayout
      title="Invalid access"
      message="No session provided. Please access this portal through your application."
    />
  );
}

/** Session ID is invalid, expired, or already exchanged. */
export function InvalidSessionError() {
  return (
    <ErrorLayout
      title="Session expired or invalid"
      message="This session has expired or is no longer valid. Please request a new session from your application."
    />
  );
}

/** Portal configuration exists but is disabled. */
export function PortalDisabledError() {
  return (
    <ErrorLayout
      title="Portal unavailable"
      message="This portal is currently unavailable. Please contact support."
    />
  );
}

/** Browser session has expired (no return_url configured). */
export function SessionExpiredError() {
  return (
    <ErrorLayout
      title="Session expired"
      message="Your session has expired. Please request a new session from your application."
    />
  );
}
