import { BrowserRouter, Routes, Route } from "react-router-dom";
import { ThemeProvider } from "./components/ThemeProvider";
import DashboardLayout from "./components/layouts/DashboardLayout";
import Endpoint from "./pages/Endpoint";
import Providers from "./pages/Providers";
import ProviderDetail from "./pages/ProviderDetail";
import Combos from "./pages/Combos";
import Usage from "./pages/Usage";
import Quota from "./pages/Quota";
import Mitm from "./pages/Mitm";
import CliTools from "./pages/CliTools";
import ConsoleLog from "./pages/ConsoleLog";
import Translator from "./pages/Translator";
import ProxyPools from "./pages/ProxyPools";
import Skills from "./pages/Skills";
import MediaProviders from "./pages/MediaProviders";
import Profile from "./pages/Profile";
import Remote from "./pages/Remote";

export default function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<DashboardLayout />}>
            <Route index element={<Endpoint />} />
            <Route path="endpoint" element={<Endpoint />} />
            <Route path="providers" element={<Providers />} />
            <Route path="providers/new" element={<Providers />} />
            <Route path="providers/:id" element={<ProviderDetail />} />
            <Route path="combos" element={<Combos />} />
            <Route path="usage" element={<Usage />} />
            <Route path="quota" element={<Quota />} />
            <Route path="mitm" element={<Mitm />} />
            <Route path="cli-tools" element={<CliTools />} />
            <Route path="cli-tools/:toolId" element={<CliTools />} />
            <Route path="console-log" element={<ConsoleLog />} />
            <Route path="translator" element={<Translator />} />
            <Route path="proxy-pools" element={<ProxyPools />} />
            <Route path="skills" element={<Skills />} />
            <Route path="media-providers/:kind" element={<MediaProviders />} />
            <Route path="media-providers/:kind/:id" element={<MediaProviders />} />
            <Route path="profile" element={<Profile />} />
            <Route path="remote" element={<Remote />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}
