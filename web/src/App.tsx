import { Routes, Route } from "react-router-dom";
import { Layout } from "@/components/Layout";
import { Dashboard } from "@/pages/Dashboard";
import { Sites } from "@/pages/Sites";
import { SiteDetail } from "@/pages/SiteDetail";
import { HistoryPage } from "@/pages/History";
import { Compare } from "@/pages/Compare";
import { Reports } from "@/pages/Reports";
import { Channels } from "@/pages/Channels";
import { Database } from "@/pages/Database";
import { NotFound } from "@/pages/NotFound";

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/sites" element={<Sites />} />
        <Route path="/sites/:id" element={<SiteDetail />} />
        <Route path="/history" element={<HistoryPage />} />
        <Route path="/compare" element={<Compare />} />
        <Route path="/reports" element={<Reports />} />
        <Route path="/channels" element={<Channels />} />
        <Route path="/database" element={<Database />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </Layout>
  );
}
