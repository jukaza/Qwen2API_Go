export const STORAGE_KEY = "qwen2api-admin-key";

export type ApiResponseEnvelope<T> = {
  ok: boolean;
  status: number;
  statusText: string;
  url: string;
  headers: Record<string, string>;
  body: T;
  rawText: string;
};

export class ApiRequestError extends Error {
  response: ApiResponseEnvelope<unknown>;

  constructor(message: string, response: ApiResponseEnvelope<unknown>) {
    super(message);
    this.name = "ApiRequestError";
    this.response = response;
  }
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
  apiKey?: string,
): Promise<T> {
  const response = await apiRequestEnvelope<T>(path, options, apiKey);
  return response.body;
}

export async function apiRequestEnvelope<T>(
  path: string,
  options: RequestInit = {},
  apiKey?: string,
): Promise<ApiResponseEnvelope<T>> {
  const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData;
  const response = await fetch(path, {
    ...options,
    headers: {
      ...(isFormData ? {} : { "Content-Type": "application/json" }),
      ...(apiKey ? { Authorization: `Bearer ${apiKey}` } : {}),
      ...(options.headers || {}),
    },
    cache: "no-store",
  });

  const text = await response.text();
  const trimmedText = text.trim();
  const isHtml = trimmedText.startsWith("<!DOCTYPE") || trimmedText.startsWith("<html") || trimmedText.startsWith("<!doctype");
  if (isHtml && (path.startsWith("/api/") || path.startsWith("/v1/") || path === "/verify")) {
    const errorEnvelope: ApiResponseEnvelope<unknown> = {
      ok: false,
      status: 404,
      statusText: "Not Found",
      url: response.url,
      headers: Object.fromEntries(response.headers.entries()),
      body: {},
      rawText: text,
    };
    throw new ApiRequestError("API endpoint not found (SPA routing fallback)", errorEnvelope);
  }

  const data = parseResponseText(text);
  const envelope: ApiResponseEnvelope<T> = {
    ok: response.ok,
    status: response.status,
    statusText: response.statusText,
    url: response.url,
    headers: Object.fromEntries(response.headers.entries()),
    body: data as T,
    rawText: text,
  };

  if (!response.ok) {
    const message =
      typeof data === "object" && data !== null
        ? extractErrorMessage(data, response.status)
        : `Yêu cầu thất bại (${response.status})`;
    throw new ApiRequestError(message, envelope as ApiResponseEnvelope<unknown>);
  }

  return envelope;
}

function parseResponseText(text: string) {
  if (!text) {
    return {};
  }

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function extractErrorMessage(data: unknown, status: number) {
  if (typeof data !== "object" || data === null) {
    return `Yêu cầu thất bại (${status})`;
  }

  const payload = data as { error?: unknown; message?: unknown };
  if (typeof payload.message === "string" && payload.message.trim()) {
    return payload.message;
  }
  if (typeof payload.error === "string" && payload.error.trim()) {
    return payload.error;
  }
  if (typeof payload.error === "object" && payload.error !== null) {
    const nested = payload.error as { message?: unknown; error?: unknown };
    if (typeof nested.message === "string" && nested.message.trim()) {
      return nested.message;
    }
    if (typeof nested.error === "string" && nested.error.trim()) {
      return nested.error;
    }
  }
  return `Yêu cầu thất bại (${status})`;
}
