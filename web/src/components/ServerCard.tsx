import { useState } from "react"
import { useNavigate } from "react-router-dom"
import type { Server } from "@/types/server"

interface ServerCardProps {
  server: Server
  org: string
  repo: string
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

export function ServerCard({ server, org, repo }: ServerCardProps) {
  const navigate = useNavigate()
  const iconSrc = server.icons?.[0]?.src
  const [imgError, setImgError] = useState(false)

  const showFallback = !iconSrc || imgError

  const handleClick = () => {
    // URL encode server name for route
    const encodedName = encodeURIComponent(server.name)
    // Pass server data via navigation state to avoid unnecessary API call
    navigate(`/${org}/${repo}/${encodedName}`, {
      state: { server, org, repo }
    })
  }

  return (
    <div 
      className="group flex flex-col bg-[#1a242a] border border-[#283339] rounded-xl overflow-hidden hover:border-primary transition-all cursor-pointer"
      onClick={handleClick}
    >
      <div className="p-5 flex-1">
        <div className="flex justify-between items-start mb-4">
          <div className="size-12 rounded-lg bg-blue-500/10 flex items-center justify-center overflow-hidden">
            {iconSrc && !imgError && (
              <img
                src={iconSrc}
                alt={server.name}
                className="size-12 object-cover rounded-lg"
                onError={() => setImgError(true)}
              />
            )}
            {showFallback && (
              <span className="material-symbols-outlined text-blue-500 text-3xl">
                extension
              </span>
            )}
          </div>
          {server.githubInfo?.stars ? (
            <div className="flex items-center gap-1 text-xs text-[#9db0b9]">
              <span className="material-symbols-outlined text-sm">star</span>
              <span>{formatStars(server.githubInfo.stars)}</span>
            </div>
          ) : null}
        </div>
        <h3 className="text-lg font-bold mb-1 group-hover:text-primary transition-colors">
          {server.name}
        </h3>
        <p className="text-sm text-[#9db0b9] mb-4 line-clamp-2">
          {server.description}
        </p>
      </div>
      <div className="px-5 py-3 border-t border-[#283339] bg-[#212b32]/50 flex justify-between items-center">
        <span className="text-xs font-mono text-slate-400">{server.version}</span>
        <span className="material-symbols-outlined text-slate-400 text-base group-hover:translate-x-1 transition-transform">
          arrow_forward
        </span>
      </div>
    </div>
  )
}
