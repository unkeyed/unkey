import { ErrorSchema } from "./http";

const content = {
  "application/json": {
    schema: ErrorSchema,
  },
};

export const openApiErrorResponses = {
  400: {
    description:
      "The server cannot or will not process the request due to something that is perceived to be a client error (e.g., malformed request syntax, invalid request message framing, or deceptive request routing).",
    content,
  },
  401: {
    description: `Although the HTTP standard specifies "unauthorized", semantically this response means "unauthenticated". That is, the client must authenticate itself to get the requested response.`,
    content,
  },
  403: {
    description:
      "The client does not have access rights to the content; that is, it is unauthorized, so the server is refusing to give the requested resource. Unlike 401 Unauthorized, the client's identity is known to the server.",
    content,
  },
  404: {
    description:
      "The server cannot find the requested resource. In the browser, this means the URL is not recognized. In an API, this can also mean that the endpoint is valid but the resource itself does not exist. Servers may also send this response instead of 403 Forbidden to hide the existence of a resource from an unauthorized client. This response code is probably the most well known due to its frequent occurrence on the web.",
  },
  405: {
    description:
      "The request method is known by the server but is not supported by the target resource. For example, an API may not allow calling DELETE to remove a resource.",
    content,
  },
  429: {
    description: `The user has sent too many requests in a given amount of time ("rate limiting")`,
    content,
  },
  500: {
    description: "The server has encountered a situation it does not know how to handle.",
    content,
  },
};
