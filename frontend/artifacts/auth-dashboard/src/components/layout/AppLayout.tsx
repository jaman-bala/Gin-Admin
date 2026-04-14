import React, { ReactNode, SyntheticEvent } from "react";
import { Link, useLocation } from "wouter";
import { useAuthStore } from "@/store/authStore";
import { useAuthLogout } from "@workspace/api-client-react";
import { useTranslation } from "@/hooks/useTranslation";
import { 
  LogOut, 
  LayoutDashboard, 
  Users, 
  Menu,
  Settings,
  Moon,
  Sun,
  ShieldCheck
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Logo } from "./Logo";
import { useTheme } from "@/hooks/use-theme";

// Compact theme toggle for sidebar
function CompactThemeToggle() {
  const { theme, setTheme } = useTheme();
  
  const toggleTheme = () => {
    if (theme === "light") setTheme("dark");
    else if (theme === "dark") setTheme("system");
    else setTheme("light");
  };
  
  return (
    <Button
      variant="ghost"
      size="icon"
      className="h-9 w-9 text-sidebar-foreground/70 hover:text-sidebar-foreground hover:bg-sidebar-accent/50"
      onClick={toggleTheme}
      title={theme === "light" ? "Светлая тема" : theme === "dark" ? "Тёмная тема" : "Системная тема"}
    >
      {theme === "light" ? <Sun className="w-4 h-4" /> : theme === "dark" ? <Moon className="w-4 h-4" /> : <Sun className="w-4 h-4 opacity-50" />}
    </Button>
  );
}

export function AppLayout({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const [location, setLocation] = useLocation();
  const { user, clearAuth, refreshToken } = useAuthStore();
  
  // DEBUG: Log user photo in sidebar
  console.log("[AppLayout] user?.photo:", user?.photo);
  
  const logoutMutation = useAuthLogout();

  const handleLogout = () => {
    if (refreshToken) {
      logoutMutation.mutate({ data: { token: refreshToken } });
    }
    clearAuth();
    setLocation("/login");
  };

  const isAdmin = user?.role === "admin" || user?.role === "superuser";

  const navItems = [
    { href: "/dashboard", label: t.common.dashboard, icon: LayoutDashboard },
    ...(isAdmin ? [
      { href: "/users", label: t.common.users, icon: Users },
      { href: "/audit", label: t.audit_page.title, icon: ShieldCheck }
    ] : []),
  ];

  const SidebarContent = () => (
    <div className="flex flex-col h-full bg-sidebar text-sidebar-foreground border-r border-sidebar-border w-64 p-4">
      <div className="mb-6 px-2">
        <Logo showText={false} size="lg" iconClassName="w-12 h-12" />
      </div>

      <nav className="flex-1 space-y-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = location === item.href;
          return (
            <Link key={item.href} href={item.href} className={`flex items-center gap-3 px-4 py-3 rounded-md transition-colors ${isActive ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium' : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground'}`}>
              <Icon className="w-5 h-5" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="mt-auto pt-4 border-t border-sidebar-border">
        {/* User Block - Clickable, goes to profile */}
        <div className="px-2 mb-3">
          <Link 
            href="/profile" 
            className={`flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors cursor-pointer ${location === '/profile' ? 'bg-sidebar-accent text-sidebar-accent-foreground' : 'hover:bg-sidebar-accent/50'}`}
          >
            <div className="relative w-9 h-9 shrink-0">
              {user?.photo ? (
                <img 
                  key={user.photo}
                  src={user.photo}
                  alt="User avatar"
                  crossOrigin="anonymous"
                  className="w-9 h-9 rounded-full object-cover border border-sidebar-border"
                  onLoad={() => console.log("[AppLayout] Sidebar avatar loaded successfully")}
                  onError={(e: SyntheticEvent<HTMLImageElement>) => {
                    console.error("[AppLayout] Sidebar avatar error:", e.currentTarget.src);
                    e.currentTarget.style.display = 'none';
                  }}
                />
              ) : (
                <div className="w-9 h-9 rounded-full bg-sidebar-primary border border-sidebar-border flex items-center justify-center">
                  <span className="text-sm text-sidebar-primary-foreground">
                    {user?.first_name?.[0] || user?.phone?.[1]}{user?.last_name?.[0] || ''}
                  </span>
                </div>
              )}
            </div>
            <div className="flex flex-col overflow-hidden flex-1 min-w-0">
              <span className="text-sm font-medium text-sidebar-foreground truncate leading-tight">
                {user?.phone}
              </span>
              <span className="text-xs text-sidebar-foreground/60 truncate capitalize">
                {user?.role === "admin" || user?.role === "superuser" ? t.common.admin : t.common.user}
              </span>
            </div>
            <Settings className="w-4 h-4 text-sidebar-foreground/40 shrink-0" />
          </Link>
        </div>

        {/* Bottom Actions Row */}
        <div className="flex items-center gap-1 px-2 mb-2">
          <CompactThemeToggle />
          <Button 
            variant="ghost" 
            size="sm"
            className="flex-1 justify-center gap-2 h-9 text-sidebar-foreground/70 hover:text-sidebar-foreground hover:bg-sidebar-accent/50"
            onClick={handleLogout}
          >
            <LogOut className="w-4 h-4" />
            <span className="text-xs">{t.common.logout}</span>
          </Button>
        </div>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-background flex flex-col md:flex-row">
      <div className="hidden md:flex flex-col w-64 fixed inset-y-0 z-50">
        <SidebarContent />
      </div>

      <div className="md:hidden flex items-center justify-between p-4 border-b border-border bg-card">
        <Logo size="sm" showText={false} />
        <Sheet>
          <SheetTrigger asChild>
            <Button variant="ghost" size="icon">
              <Menu className="w-6 h-6" />
            </Button>
          </SheetTrigger>
          <SheetContent side="left" className="p-0 w-64 border-r-0">
            <SidebarContent />
          </SheetContent>
        </Sheet>
      </div>

      <main className="flex-1 md:pl-64 flex flex-col">
        <div className="flex-1 p-4 md:p-8 max-w-6xl mx-auto w-full">
          {children}
        </div>
      </main>
    </div>
  );
}
