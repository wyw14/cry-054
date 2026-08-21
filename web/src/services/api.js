export class ApiError extends Error {
    status;
    body;
    constructor(status, body) {
        super(body.message);
        this.status = status;
        this.body = body;
    }
}
export async function api(path, init = {}) {
    const headers = new Headers(init.headers);
    headers.set('Accept', 'application/json');
    if (init.body)
        headers.set('Content-Type', 'application/json');
    headers.set('X-Actor-ID', 'admin-demo');
    const response = await fetch(`/api/v1${path}`, { ...init, headers });
    const body = await response.json();
    if (!response.ok)
        throw new ApiError(response.status, body);
    return body;
}
