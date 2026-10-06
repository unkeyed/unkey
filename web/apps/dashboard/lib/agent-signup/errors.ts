export class AgentSignupError extends Error {
  readonly status: number;
  readonly code: string;
  readonly authenticate: boolean;

  constructor(status: number, code: string, message: string, authenticate = false) {
    super(message);
    this.name = "AgentSignupError";
    this.status = status;
    this.code = code;
    this.authenticate = authenticate;
  }
}
