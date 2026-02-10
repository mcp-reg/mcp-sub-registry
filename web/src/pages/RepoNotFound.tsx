import { useNavigate } from "react-router-dom"
import { Header } from "@/components/Header"
import { Footer } from "@/components/Footer"
import { Button } from "@/components/ui/button"

export function RepoNotFound() {
  const navigate = useNavigate()
  const templateUrl = "https://github.com/mcp-reg/mcp-registry-template/generate"

  return (
    <div className="dark min-h-screen bg-background-dark text-slate-100 font-display flex flex-col">
      <Header />
      <main className="flex-1 flex items-center justify-center px-6">
        <div className="max-w-lg w-full bg-[#1a242a] border border-[#283339] rounded-xl p-6 shadow-sm">
          <div className="text-center">
            {/* Icon */}
            <div className="w-16 h-16 rounded-full bg-red-500/10 flex items-center justify-center mx-auto mb-4">
              <span className="material-symbols-outlined text-red-400 text-3xl">
                dns
              </span>
            </div>

            {/* Title */}
            <h2 className="text-xl font-semibold mb-2">Registry Not Found</h2>

            {/* Message */}
            <p className="text-slate-400 mb-6">
              Could not find registry.json in this repository. Make sure you've forked the mcp-registry-template template.
            </p>

            {/* Buttons */}
            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              <Button
                variant="outline"
                onClick={() => navigate("/")}
                className="border-slate-700 hover:bg-slate-800"
              >
                <span className="material-symbols-outlined mr-2 text-lg">arrow_back</span>
                Go Back
              </Button>
              <a
                href={templateUrl}
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button className="w-full">
                  Use Template
                  <span className="material-symbols-outlined ml-2 text-lg">open_in_new</span>
                </Button>
              </a>
            </div>
          </div>
        </div>
      </main>
      <Footer />
    </div>
  )
}
