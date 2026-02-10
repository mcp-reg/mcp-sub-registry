import { useState, useMemo } from "react"
import { Header } from "@/components/Header"
import { Footer } from "@/components/Footer"
import { ServerCard } from "@/components/ServerCard"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import type { ServerWrapper } from "@/types/server"

interface RegistryProps {
  org: string
  repo: string
  branch: string
  servers: ServerWrapper[]
  serverCount: number
  refreshing: boolean
  copied: boolean
  onRefresh: () => void
  onCopyUrl: () => void
  onViewGithub: () => void
}

export function Registry({
  org,
  repo,
  branch: _branch,
  servers,
  serverCount,
  refreshing,
  copied,
  onRefresh,
  onCopyUrl,
  onViewGithub,
}: RegistryProps) {
  const [search, setSearch] = useState("")

  const filteredServers = useMemo(() => {
    if (!search.trim()) return servers
    const query = search.toLowerCase()
    return servers.filter(({ server }) =>
      server.name.toLowerCase().includes(query) ||
      server.description.toLowerCase().includes(query)
    )
  }, [servers, search])

  return (
    <div className="dark min-h-screen bg-background-dark text-slate-100 font-display flex flex-col">
      <Header />

      {/* Registry Header Bar */}
      <div className="w-full bg-[#1a242a] border-b border-[#283339]">
        <div className="max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between flex-wrap gap-4">
            {/* Left: org/repo + count */}
            <div className="flex items-center gap-3">
              <span className="material-symbols-outlined text-primary text-2xl">inventory_2</span>
              <div>
                <h1 className="text-xl font-bold tracking-tight">
                  {org}/{repo}
                </h1>
                <p className="text-sm text-[#9db0b9]">
                  {serverCount} MCP server{serverCount !== 1 ? "s" : ""}
                </p>
              </div>
            </div>

            {/* Right: Action buttons */}
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={onRefresh}
                disabled={refreshing}
                className="gap-2"
              >
                <span className={`material-symbols-outlined text-lg ${refreshing ? "animate-spin" : ""}`}>
                  refresh
                </span>
                {refreshing ? "Refreshing..." : "Refresh"}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={onCopyUrl}
                className="gap-2"
              >
                <span className="material-symbols-outlined text-lg">
                  {copied ? "check" : "content_copy"}
                </span>
                {copied ? "Copied!" : "Copy API URL"}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={onViewGithub}
                className="gap-2"
              >
                <span className="material-symbols-outlined text-lg">open_in_new</span>
                View on GitHub
              </Button>
            </div>
          </div>
        </div>
      </div>

      <main className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-10">
        {/* Search */}
        <div className="flex flex-col gap-6 mb-12">
          <div className="relative w-full">
            <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
              <span className="material-symbols-outlined text-[#9db0b9]">search</span>
            </div>
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="block w-full pl-12 pr-4 py-4 bg-[#283339] border-none rounded-xl text-lg focus:ring-2 focus:ring-primary placeholder:text-[#9db0b9] transition-all h-auto"
              placeholder="Search servers by name or description..."
            />
          </div>
        </div>

        {/* Server Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {filteredServers.map((wrapper) => (
            <ServerCard
              key={wrapper.server.name}
              server={wrapper.server}
              org={org}
              repo={repo}
            />
          ))}
          {servers.length === 0 && (
            <div className="col-span-full text-center py-12 text-slate-500">
              No servers configured in this registry yet.
            </div>
          )}
          {servers.length > 0 && filteredServers.length === 0 && (
            <div className="col-span-full text-center py-12 text-slate-500">
              No servers match your search.
            </div>
          )}
        </div>
      </main>
      <Footer />
    </div>
  )
}
