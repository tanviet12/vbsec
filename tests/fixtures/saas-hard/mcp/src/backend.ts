export class BackendError extends Error {
  constructor(public status: number) {
    super(`backend responded with ${status}`);
  }
}

// Backend talks to the Tallyboard API as the member who created the
// assistant token. Every call carries that member's rights and company.
export class Backend {
  constructor(
    private readonly baseUrl: string,
    private readonly token: string,
  ) {}

  private async request<T>(method: string, path: string, body?: unknown, headers: Record<string, string> = {}): Promise<T> {
    const res = await fetch(new URL(path, this.baseUrl), {
      method,
      headers: {
        Authorization: `Bearer ${this.token}`,
        ...(body === undefined ? {} : { "Content-Type": "application/json" }),
        ...headers,
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.timeout(15_000),
    });
    if (!res.ok) {
      throw new BackendError(res.status);
    }
    if (res.status === 204) {
      return undefined as T;
    }
    return (await res.json()) as T;
  }

  get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body ?? {});
  }

  put<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>("PUT", path, body);
  }

  del<T>(path: string, headers?: Record<string, string>): Promise<T> {
    return this.request<T>("DELETE", path, undefined, headers);
  }
}

export function seg(value: string | number): string {
  return encodeURIComponent(String(value));
}

export function backendFromEnv(): Backend {
  const baseUrl = process.env.TALLYBOARD_URL;
  const token = process.env.TALLYBOARD_TOKEN;
  if (!baseUrl || !token) {
    throw new Error("TALLYBOARD_URL and TALLYBOARD_TOKEN must be set");
  }
  return new Backend(baseUrl, token);
}
