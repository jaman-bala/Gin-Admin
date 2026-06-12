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
import { Logo } from "./Logo";
import { useTheme } from "@/hooks/use-theme";

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
    <div className="flex flex-col h-full bg-sidebar text-sidebar-foreground border-r border-sidebar-border w-64">
      {/* Logo Header */}
      <div className="px-5 py-5 border-b border-sidebar-border/60">
        <Logo showText size="sm" />
      </div>

      {/* Nav */}
      <nav className="flex-1 px-3 py-4 space-y-0.5">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = location === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                isActive
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground"
              }`}
            >
              <Icon className="w-4 h-4 shrink-0" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="px-3 pb-4 pt-3 border-t border-sidebar-border/60 space-y-1">
        <Link
          href="/profile"
          className={`flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-150 cursor-pointer ${
            location === "/profile"
              ? "bg-sidebar-accent text-sidebar-accent-foreground"
              : "hover:bg-sidebar-accent/60"
          }`}
        >
          <div className="w-8 h-8 shrink-0">
            {user?.photo ? (
              <img
                key={user.photo}
                src={user.photo}
                alt="avatar"
                crossOrigin="anonymous"
                className="w-8 h-8 rounded-full object-cover border border-sidebar-border"
                onError={(e: SyntheticEvent<HTMLImageElement>) => { e.currentTarget.style.display = "none"; }}
              />
            ) : (
              <div className="w-8 h-8 rounded-full bg-primary/20 border border-primary/30 flex items-center justify-center">
                <span className="text-xs font-semibold text-primary">
                  {user?.first_name?.[0] || user?.phone?.[1]}{user?.last_name?.[0] || ""}
                </span>
              </div>
            )}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-sidebar-foreground truncate leading-tight">
              {user?.first_name ? `${user.first_name} ${user.last_name || ""}`.trim() : user?.phone}
            </p>
            <p className="text-xs text-sidebar-foreground/50 capitalize">
              {user?.role === "superuser" ? "Суперадмин" : user?.role === "admin" ? t.common.admin : t.common.user}
            </p>
          </div>
          <Settings className="w-3.5 h-3.5 text-sidebar-foreground/30 shrink-0" />
        </Link>

        <div className="flex items-center gap-1">
          <CompactThemeToggle />
          <Button
            variant="ghost"
            size="sm"
            className="flex-1 justify-start gap-2 h-9 text-sidebar-foreground/60 hover:text-sidebar-foreground hover:bg-sidebar-accent/50 text-xs"
            onClick={handleLogout}
          >
            <LogOut className="w-4 h-4" />
            {t.common.logout}
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
        <Logo size="sm" showText />
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

      <main className="flex-1 md:pl-64 min-h-screen flex flex-col">
        <div className="flex-1 p-6 md:p-8 max-w-6xl mx-auto w-full">
          {children}
        </div>
      </main>
    </div>
  );
}