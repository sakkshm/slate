export interface APIError {
  status: number;
  code: string;
  message: string;
}

const apiBase = () => import.meta.env.VITE_API_URL as string;

const UNAUTHORIZED_EVENT = "slate:unauthorized";

let unauthorizedNotified = false;

export function notifyUnauthorized(): void {
  if (unauthorizedNotified) return;
  unauthorizedNotified = true;
  window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
}

export function onUnauthorized(handler: () => void): () => void {
  const listener = () => handler();
  window.addEventListener(UNAUTHORIZED_EVENT, listener);
  return () => window.removeEventListener(UNAUTHORIZED_EVENT, listener);
}

async function request<T>(endpoint: string, method: string, body?: unknown): Promise<T> {
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
  const url = `${apiBase()}/${cleanEndpoint}`;

  let response: Response;

  try {
    response = await fetch(url, {
      method,
      headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      credentials: 'include',
    });
  } catch (error) {
    throw {
      status: 0,
      code: 'NETWORK_ERROR',
      message: error instanceof Error ? error.message : 'Network connectivity failure.',
    } as APIError;
  }

  if (!response.ok) {
    let code = 'UNKNOWN_ERROR';
    let message = `HTTP Error ${response.status}`;

    try {
      const errorData = await response.json();
      if (errorData?.code) code = errorData.code;
      if (errorData?.msg) message = errorData.msg;
      if (errorData?.message) message = errorData.message;
    } catch {
      message = response.statusText || message;
    }

    if (response.status === 401 && code === "UNAUTHORIZED") {
      notifyUnauthorized();
    }

    throw { status: response.status, code, message } as APIError;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  try {
    return (await response.json()) as T;
  } catch {
    throw {
      status: response.status,
      code: 'MALFORMED_JSON',
      message: 'The server responded successfully, but the payload could not be parsed.',
    } as APIError;
  }
}

export const apiClient = {
  get: <T>(endpoint: string): Promise<T> => request<T>(endpoint, 'GET'),

  post: <T>(endpoint: string, body?: unknown): Promise<T> => request<T>(endpoint, 'POST', body),

  put: <T>(endpoint: string, body?: unknown): Promise<T> => request<T>(endpoint, 'PUT', body),

  del: <T>(endpoint: string): Promise<T> => request<T>(endpoint, 'DELETE'),
};

export interface SSEHandlers {
  onLine: (line: string) => void;
  onDone?: () => void;
  onError?: () => void;
}

export function subscribeSSE(endpoint: string, handlers: SSEHandlers): () => void {
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
  const url = `${apiBase()}/${cleanEndpoint}`;

  const source = new EventSource(url, { withCredentials: true });

  let completed = false;

  source.onmessage = (event) => {
    if (completed) return;
    handlers.onLine(event.data as string);
  };

  source.addEventListener('done', () => {
    if (completed) return;
    completed = true;
    handlers.onDone?.();
    source.close();
  });

  source.onerror = () => {
    if (completed) return;
    handlers.onError?.();
  };

  return () => {
    if (!completed) {
      completed = true;
      source.close();
    }
  };
}
