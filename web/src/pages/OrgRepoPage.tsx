import { useEffect, useState, useCallback } from "react"
import { useParams, useSearchParams } from "react-router-dom"
import { RepoNotFound } from "./RepoNotFound"
import { Registry } from "./Registry"
import { fetchServers, refreshCache } from "@/api/registry"
import type { ServerWrapper } from "@/types/server"

type LoadingState = "loading" | "success" | "error"

export function OrgRepoPage() {
  const { org, repo } = useParams<{ org: string; repo: string }>()
  const [searchParams] = useSearchParams()
  const branch = searchParams.get("branch") || "main"

  const [state, setState] = useState<LoadingState>("loading")
  const [servers, setServers] = useState<ServerWrapper[]>([])
  const [refreshing, setRefreshing] = useState(false)
  const [copied, setCopied] = useState(false)

  const loadServers = useCallback(async () => {
    if (!org || !repo) {
      setState("error")
      return
    }

    setState("loading")
    try {
      const data = await fetchServers(org, repo, branch)
      setServers(data.servers)
      setState("success")
    } catch {
      setState("error")
    }
  }, [org, repo, branch])

  useEffect(() => {
    loadServers()
  }, [loadServers])

  const handleRefresh = async () => {
    if (!org || !repo || refreshing) return
    setRefreshing(true)
    try {
      await refreshCache(org, repo, branch)
      await loadServers()
    } finally {
      setRefreshing(false)
    }
  }

  const handleCopyUrl = async () => {
    if (!org || !repo) return
    const apiUrl = `${window.location.origin}/${org}/${repo}/${branch}/v0.1/servers`
    await navigator.clipboard.writeText(apiUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleViewGithub = () => {
    if (!org || !repo) return
    window.open(`https://github.com/${org}/${repo}`, "_blank")
  }

  if (state === "loading") {
    return (
      <div className="dark min-h-screen bg-background-dark flex items-center justify-center">
        <div className="text-primary text-xl">Loading...</div>
      </div>
    )
  }

  if (state === "error") {
    return <RepoNotFound />
  }

  return (
    <Registry
      org={org!}
      repo={repo!}
      branch={branch}
      servers={servers}
      serverCount={servers.length}
      refreshing={refreshing}
      copied={copied}
      onRefresh={handleRefresh}
      onCopyUrl={handleCopyUrl}
      onViewGithub={handleViewGithub}
    />
  )
}
