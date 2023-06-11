import { HTTPException } from "hono/http-exception";

export class AuthorizationError extends HTTPException {
  constructor(message: string) {
    super(403, { message });
  }
}

export class NotFoundError extends HTTPException {
  constructor(message: string) {
    super(404, { message });
  }
}

export class InternalServerError extends HTTPException {
  constructor(message: string) {
    super(500, { message });
  }
}

export class BadRequestError extends HTTPException {
  constructor(message: string) {
    super(400, { message });
  }
}
