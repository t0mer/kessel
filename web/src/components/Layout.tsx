import { useState, type ReactNode } from "react";
import { NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Globe,
  History,
  GitCompareArrows,
  FileText,
  Bell,
  Database,
  Moon,
  Sun,
  Menu,
  X,
  Gauge as GaugeIcon,
} from "lucide-react";
import { useTheme } from "@/lib/hooks";

const NAV = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "/sites", label: "Sites", icon: Globe },
  { to: "/history", label: "History", icon: History },
  { to: "/compare", label: "Compare", icon: GitCompareArrows },
  { to: "/reports", label: "Reports", icon: FileText },
  { to: "/channels", label: "Channels", icon: Bell },
  { to: "/database", label: "Database", icon: Database },
];

function NavItems({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="space-y-1">
      {NAV.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          onClick={onNavigate}
          className={({ isActive }) =>
            [
              "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
              isActive ? "bg-primary/15 text-primary" : "text-muted hover:bg-surface-2 hover:text-fg",
            ].join(" ")
          }
        >
          <Icon size={18} />
          {label}
        </NavLink>
      ))}
    </nav>
  );
}

function Brand() {
  return (
    <div className="flex items-center gap-2.5">
      <span className="grid h-9 w-9 place-items-center rounded-lg bg-primary/15 text-primary">
        <GaugeIcon size={20} />
      </span>
      <div className="leading-tight">
        <div className="font-display text-lg font-bold text-fg">Kessel</div>
        <div className="text-[10px] uppercase tracking-widest text-muted">speed telemetry</div>
      </div>
    </div>
  );
}

function ThemeButton() {
  const [dark, toggle] = useTheme();
  return (
    <button
      onClick={toggle}
      aria-label={dark ? "Switch to light mode" : "Switch to dark mode"}
      className="grid h-9 w-9 place-items-center rounded-lg border border-border text-muted hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      {dark ? <Sun size={18} /> : <Moon size={18} />}
    </button>
  );
}

export function Layout({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="min-h-full">
      {/* Desktop sidebar */}
      <aside className="fixed inset-y-0 left-0 hidden w-64 flex-col border-r border-border bg-surface p-4 lg:flex">
        <div className="mb-8">
          <Brand />
        </div>
        <NavItems />
        <div className="mt-auto flex items-center justify-between">
          <span className="text-xs text-muted">No login · LAN</span>
          <ThemeButton />
        </div>
      </aside>

      {/* Mobile top bar */}
      <header className="sticky top-0 z-30 flex items-center justify-between border-b border-border bg-surface/90 px-4 py-3 backdrop-blur lg:hidden">
        <Brand />
        <div className="flex items-center gap-2">
          <ThemeButton />
          <button
            onClick={() => setOpen(true)}
            aria-label="Open navigation"
            className="grid h-9 w-9 place-items-center rounded-lg border border-border text-muted hover:text-fg"
          >
            <Menu size={18} />
          </button>
        </div>
      </header>

      {/* Mobile drawer */}
      {open && (
        <div className="fixed inset-0 z-40 bg-black/50 lg:hidden" onClick={() => setOpen(false)}>
          <div className="absolute inset-y-0 left-0 w-72 bg-surface p-4" onClick={(e) => e.stopPropagation()}>
            <div className="mb-6 flex items-center justify-between">
              <Brand />
              <button onClick={() => setOpen(false)} aria-label="Close navigation" className="text-muted hover:text-fg">
                <X size={20} />
              </button>
            </div>
            <NavItems onNavigate={() => setOpen(false)} />
          </div>
        </div>
      )}

      <main className="px-4 py-6 lg:ml-64 lg:px-8 lg:py-8">
        <div className="mx-auto max-w-5xl">{children}</div>
      </main>
    </div>
  );
}

export function PageHeader({ title, subtitle, action }: { title: string; subtitle?: string; action?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="font-display text-2xl font-bold text-fg">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-muted">{subtitle}</p>}
      </div>
      {action}
    </div>
  );
}
