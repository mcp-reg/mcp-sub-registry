import { useState, useEffect } from "react"
import { useParams, useNavigate, useLocation } from "react-router-dom"
import ReactMarkdown from 'react-markdown'
import rehypeRaw from 'rehype-raw'
import remarkGfm from 'remark-gfm'
import { Header } from "@/components/Header"
import { Footer } from "@/components/Footer"
import type { Server } from "@/types/server"

interface LocationState {
  server?: Server
  org?: string
  repo?: string
}

function formatStars(count: number): string {
  if (count >= 1000000) {
    return (count / 1000000).toFixed(1).replace(/\.0$/, '') + 'M'
  }
  if (count >= 1000) {
    return (count / 1000).toFixed(1).replace(/\.0$/, '') + 'k'
  }
  return count.toString()
}

export function ServerDetailPage() {
  const { org, repo } = useParams<{ org: string; repo: string; serverName: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const locationState = location.state as LocationState | null
  
  // Use server data from navigation state (no API call needed)
  const server = locationState?.server || null
  const [imgError, setImgError] = useState(false)
  const [fetchedReadme, setFetchedReadme] = useState<string | null>(null)
  
  // Note: Server data is always passed from Registry page via navigation state.
  // If no server data is available, show error state.

  // Fetch README from GitHub if not provided by server
  useEffect(() => {
    // Only fetch if no readme exists but repository does
    if (server?.readme || !server?.repository?.url) {
      return
    }

    // Only fetch from GitHub repositories
    const repoUrl = server.repository.url.toLowerCase()
    if (!repoUrl.includes('github.com')) {
      return
    }

    // Parse owner/repo from URL
    const match = server.repository.url.match(/github\.com\/([^/]+)\/([^/]+)/i)
    if (!match) {
      return
    }

    const [, owner, repo] = match
    const cleanRepo = repo.replace(/\.git$/, '')
    const subfolder = server.repository.subfolder

    // Build raw GitHub URL
    const basePath = subfolder ? `${subfolder}/` : ''
    const rawUrl = `https://raw.githubusercontent.com/${owner}/${cleanRepo}/HEAD/${basePath}README.md`

    fetch(rawUrl)
      .then(res => {
        if (!res.ok) throw new Error('Not found')
        return res.text()
      })
      .then(text => setFetchedReadme(text))
      .catch(() => {
        // Silently fail - no README to display
        setFetchedReadme(null)
      })
  }, [server])

  // Compute effective readme and GitHub URL
  const effectiveReadme = server?.readme || fetchedReadme
  const effectiveGithubUrl = server?.githubInfo?.url || 
    (server?.repository?.url?.toLowerCase().includes('github.com') ? server.repository.url : null)

  const handleBack = () => {
    // Navigate back to registry, preserving org/repo from params or state
    const backOrg = org || locationState?.org
    const backRepo = repo || locationState?.repo
    if (backOrg && backRepo) {
      navigate(`/${backOrg}/${backRepo}`)
    }
  }

  if (!server) {
    return (
      <div className="dark min-h-screen bg-background-dark text-slate-100 font-display flex flex-col">
        <Header />
        <main className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-10">
          <div className="text-center">
            <h1 className="text-2xl font-bold mb-4">Server Not Found</h1>
            <p className="text-slate-400 mb-6">The requested server could not be found.</p>
            <button
              onClick={handleBack}
              className="inline-flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary/90 text-white rounded-lg transition-colors"
            >
              <span className="material-symbols-outlined">arrow_back</span>
              Back to Registry
            </button>
          </div>
        </main>
        <Footer />
      </div>
    )
  }

  const iconSrc = server.icons?.[0]?.src
  const showFallback = !iconSrc || imgError

  return (
    <div className="dark min-h-screen bg-background-dark text-slate-100 font-display flex flex-col">
      <Header />
      <main className="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-10">
        {/* Back Button */}
        <button
          onClick={handleBack}
          className="mb-6 inline-flex items-center gap-2 text-slate-400 hover:text-primary transition-colors"
        >
          <span className="material-symbols-outlined">arrow_back</span>
          <span>Back to Registry</span>
        </button>

        {/* Server Header */}
        <div className="mb-8">
          <div className="flex items-start gap-6 mb-6">
            <div className="size-16 rounded-lg bg-blue-500/10 flex items-center justify-center overflow-hidden shrink-0">
              {iconSrc && !imgError && (
                <img
                  src={iconSrc}
                  alt={server.title || server.name}
                  className="size-16 object-cover rounded-lg"
                  onError={() => setImgError(true)}
                />
              )}
              {showFallback && (
                <span className="material-symbols-outlined text-blue-500 text-5xl">
                  extension
                </span>
              )}
            </div>
            <div className="flex-1">
              <h1 className="text-4xl md:text-5xl font-black tracking-tight mb-2">
                {server.title || server.name}
              </h1>
              <p className="text-lg text-[#9db0b9] mb-4">
                {server.description}
              </p>
              <div className="flex items-center gap-4 flex-wrap">
                <span className="text-sm font-mono text-slate-400">v{server.version}</span>
                {server.githubInfo?.stars && (
                  <div className="flex items-center gap-1 text-sm text-[#9db0b9]">
                    <span className="material-symbols-outlined text-base">star</span>
                    <span>{formatStars(server.githubInfo.stars)}</span>
                  </div>
                )}
                {effectiveGithubUrl && (
                  <a
                    href={effectiveGithubUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-sm text-primary hover:text-primary/80 transition-colors"
                  >
                    <span className="material-symbols-outlined text-base">open_in_new</span>
                    <span>View on GitHub</span>
                  </a>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* GitHub Info Card */}
        {server.githubInfo && (
          <div className="bg-[#1a242a] border border-[#283339] rounded-xl p-6 mb-8">
            <h2 className="text-xl font-bold mb-4">GitHub Repository</h2>
            <div className="space-y-2">
              {server.githubInfo.owner && server.githubInfo.repo && (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-slate-400">Repository:</span>
                  <span className="text-sm font-mono text-slate-100">
                    {server.githubInfo.owner}/{server.githubInfo.repo}
                  </span>
                </div>
              )}
              {server.githubInfo.stars && (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-slate-400">Stars:</span>
                  <span className="text-sm text-slate-100">{server.githubInfo.stars.toLocaleString()}</span>
                </div>
              )}
              {server.githubInfo.contributors && (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-slate-400">Contributors:</span>
                  <span className="text-sm text-slate-100">{server.githubInfo.contributors}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* README Section */}
        {effectiveReadme && (
          <div className="bg-[#1a242a] border border-[#283339] rounded-xl p-6">
            <h2 className="text-xl font-bold mb-4">README</h2>
            <div className="prose prose-invert max-w-none prose-headings:text-slate-100 prose-p:text-slate-300 prose-a:text-primary prose-a:no-underline hover:prose-a:underline prose-strong:text-slate-100 prose-code:text-primary prose-code:bg-[#283339] prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-pre:bg-[#212b32] prose-pre:border prose-pre:border-[#283339]">
              <ReactMarkdown 
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeRaw]}
              >
                {effectiveReadme}
              </ReactMarkdown>
            </div>
          </div>
        )}
      </main>
      <Footer />
    </div>
  )
}
