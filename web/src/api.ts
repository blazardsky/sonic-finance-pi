// One fetch wrapper for every /api/ call. It throws the Response itself on a
// non-2xx, which is what lets a caller tell a refusal (409, 400) apart from a
// Pi that has gone away, without a bespoke error type.
export async function api(path: string, init?: RequestInit) {
  const res = await fetch(path, {
    ...init,
    headers: init?.body ? { "Content-Type": "application/json" } : undefined,
  })
  if (!res.ok) throw res
  return res
}

export async function apiJSON<T>(path: string, init?: RequestInit): Promise<T> {
  return (await api(path, init)).json() as Promise<T>
}
