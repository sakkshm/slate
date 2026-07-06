export interface APIError {
  status: number;
  code: string;
  message: string;
}

export const apiClient = {
  get: async <T>(endpoint: string): Promise<T> => {
    const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
    const url = `${import.meta.env.VITE_API_URL}/${cleanEndpoint}`;

    let response: Response;

    try {
      response = await fetch(url, {
        method: 'GET',
        // CRITICAL: Tells the browser to accept and store cookies from cross-origin requests
        credentials: 'include', 
      });
    } catch (networkError: any) {
      throw {
        status: 0, 
        code: 'NETWORK_ERROR',
        message: networkError?.message || 'Network connectivity failure.',
      } as APIError;
    }

    if (!response.ok) {
      let code = 'UNKNOWN_ERROR';
      let message = `HTTP Error ${response.status}`;

      try {
        const errorData = await response.json();
        if (errorData?.code) code = errorData.code;
        if (errorData?.message) message = errorData.message;
      } catch {
        message = response.statusText || message;
      }

      throw { status: response.status, code, message } as APIError;
    }

    try {
      return await response.json();
    } catch (parseError: any) {
      throw {
        status: response.status,
        code: 'MALFORMED_JSON',
        message: 'The server responded successfully, but the payload could not be parsed.',
      } as APIError;
    }
  },

  post: async <T>(endpoint: string, body?: any): Promise<T> => {
    const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
    const url = `${import.meta.env.VITE_API_URL}/${cleanEndpoint}`;

    let response: Response;

    try {
      response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: body ? JSON.stringify(body) : undefined,
        // CRITICAL: Send existing cookies up, and allow incoming ones to be saved
        credentials: 'include', 
      });
    } catch (networkError: any) {
      throw {
        status: 0,
        code: 'NETWORK_ERROR',
        message: networkError?.message || 'Network connectivity failure.',
      } as APIError;
    }

    if (!response.ok) {
      let code = 'UNKNOWN_ERROR';
      let message = `HTTP Error ${response.status}`;

      try {
        const errorData = await response.json();
        if (errorData?.code) code = errorData.code;
        if (errorData?.message) message = errorData.message;
      } catch {
        message = response.statusText || message;
      }

      throw { status: response.status, code, message } as APIError;
    }

    try {
      return await response.json();
    } catch (parseError: any) {
      throw {
        status: response.status,
        code: 'MALFORMED_JSON',
        message: 'The server responded successfully, but the payload could not be parsed.',
      } as APIError;
    }
  }
};