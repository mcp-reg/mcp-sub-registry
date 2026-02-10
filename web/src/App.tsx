import { BrowserRouter, Routes, Route } from "react-router-dom"
import { SetupJourney } from "@/pages/SetupJourney"
import { OrgRepoPage } from "@/pages/OrgRepoPage"
import { ServerDetailPage } from "@/pages/ServerDetailPage"

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<SetupJourney />} />
        <Route path="/:org/:repo" element={<OrgRepoPage />} />
        <Route path="/:org/:repo/:serverName" element={<ServerDetailPage />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
