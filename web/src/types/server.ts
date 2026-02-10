export interface Icon {
  src: string
  mimeType?: string
  sizes?: string
  theme?: string
}

export interface GithubInfo {
  owner?: string
  repo?: string
  url?: string
  stars?: number
  contributors?: number
}

export interface Repository {
  url?: string
  source?: string
  id?: string
  subfolder?: string
}

export interface Server {
  name: string
  title?: string
  description: string
  version: string
  icons?: Icon[]
  githubInfo?: GithubInfo
  repository?: Repository
  readme?: string
}

export interface ServerWrapper {
  server: Server
  _meta?: Record<string, unknown>
}

export interface ServersResponse {
  servers: ServerWrapper[]
  metadata: {
    nextCursor: string | null
    count: number
  }
}

export interface ErrorResponse {
  error: string
  docs?: string
}

export interface VersionResponse {
  server: Server
  _meta?: Record<string, unknown>
}
