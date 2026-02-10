import type { ServersResponse, ErrorResponse, VersionResponse } from "@/types/server"

const API_BASE = import.meta.env.VITE_API_BASE || ""

export async function fetchServers(
  org: string,
  repo: string,
  branch: string = "main"
): Promise<ServersResponse> {
  const url = `${API_BASE}/${org}/${repo}/${branch}/v0.1/servers?limit=1000`
  const response = await fetch(url)

  if (!response.ok) {
    const error: ErrorResponse = await response.json().catch(() => ({
      error: "Failed to fetch servers"
    }))
    throw new Error(error.error)
  }

  return response.json()
}

export async function refreshCache(
  org: string,
  repo: string,
  branch: string = "main"
): Promise<void> {
  const url = `${API_BASE}/${org}/${repo}/${branch}/v0.1/refresh`
  const response = await fetch(url, { method: "POST" })
  if (!response.ok) throw new Error("Failed to refresh cache")
}

export async function fetchServerDetails(
  org: string,
  repo: string,
  serverName: string,
  branch: string = "main"
): Promise<VersionResponse> {
  // URL encode server name (handles slashes like "ai.exa/exa" → "ai.exa%2Fexa")
  const encodedServerName = encodeURIComponent(serverName)
  const url = `${API_BASE}/${org}/${repo}/${branch}/v0.1/servers/${encodedServerName}/versions/latest`
  const response = await fetch(url)

  if (!response.ok) {
    const error: ErrorResponse = await response.json().catch(() => ({
      error: "Failed to fetch server details"
    }))
    throw new Error(error.error)
  }

  return response.json()
}
