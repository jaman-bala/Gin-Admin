import { useEffect } from "react";
import { 
  useGetMe, 
  getGetMeQueryKey, 
  useGetUserStats, 
  getGetUserStatsQueryKey,
  useGetAuditLogs,
  getGetAuditLogsQueryKey
} from "@workspace/api-client-react";
import { Link } from "wouter";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import { useAuthStore } from "@/store/authStore";
import { useTranslation } from "@/hooks/useTranslation";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { 
  Activity, 
  Users, 
  Shield, 
  UserPlus, 
  LucideIcon, 
  ListChecks, 
  ArrowUpRight, 
  Clock 
} from "lucide-react";

export default function Dashboard() {
  const { t } = useTranslation();
  const { user: storeUser, setUser } = useAuthStore();
  
  const { data: me, isLoading: meLoading } = useGetMe({ 
    query: { 
      queryKey: getGetMeQueryKey(),
      refetchOnWindowFocus: false,
    } 
  });

  useEffect(() => {
    if (me) {
      setUser(me);
    }
  }, [me, setUser]);

  const currentUser = me || storeUser;
  const isAdmin = currentUser?.role === "admin" || currentUser?.role === "superuser";

  const { data: stats, isLoading: statsLoading } = useGetUserStats({
    query: {
      queryKey: getGetUserStatsQueryKey(),
      enabled: !!currentUser && isAdmin,
    }
  });

  const { data: auditLogs, isLoading: auditLoading } = useGetAuditLogs({
    query: {
      queryKey: getGetAuditLogsQueryKey(),
      enabled: !!currentUser && isAdmin,
      select: (data) => data.slice(0, 5),
    }
  });

  if (meLoading && !currentUser) {
    return <DashboardSkeleton />;
  }

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t.common.dashboard}</h1>
        <p className="text-muted-foreground mt-1">{t.dashboard.system_overview}</p>
      </div>

      <div className="space-y-6">
        {isAdmin && (
          <div className="space-y-4">
            <h3 className="text-lg font-medium tracking-tight">{t.dashboard.system_stats}</h3>
            {statsLoading ? (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {[1, 2, 3, 4].map(i => <Skeleton key={i} className="h-28 rounded-xl" />)}
              </div>
            ) : stats ? (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <StatCard title={t.users_page.total_users} value={stats.total} icon={Users} trend="+12%" />
                <StatCard title={t.common.active} value={stats.active} icon={Activity} color="text-emerald-500" />
                <StatCard title={t.dashboard.admins} value={stats.admins} icon={Shield} color="text-primary" />
                <StatCard title={t.dashboard.new_month} value={stats.new_this_month} icon={UserPlus} />
              </div>
            ) : null}
          </div>
        )}

        <Card className="border-border shadow-sm">
          <CardHeader>
            <CardTitle>{t.dashboard.welcome_title}</CardTitle>
            <CardDescription>{t.dashboard.welcome_description}</CardDescription>
          </CardHeader>
          <CardContent className="prose prose-sm dark:prose-invert max-w-none">
            <p>
              {t.dashboard.platform_info} 
              {isAdmin ? ` ${t.dashboard.admin_access}` : ` ${t.dashboard.user_access}`}
            </p>
            {!isAdmin && (
              <div className="mt-4 p-4 bg-muted/50 rounded-lg border border-border">
                <h4 className="font-medium text-foreground m-0 mb-2">{t.dashboard.need_help}</h4>
                <p className="text-muted-foreground text-sm m-0">{t.dashboard.contact_admin}</p>
              </div>
            )}
          </CardContent>
        </Card>

        {isAdmin && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card className="border-border shadow-sm overflow-hidden">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
                <div className="space-y-1">
                  <CardTitle className="text-xl flex items-center gap-2">
                    <ListChecks className="w-5 h-5 text-primary" />
                    Недавняя активность
                  </CardTitle>
                  <CardDescription>Последние события в системе</CardDescription>
                </div>
                <Link href="/audit">
                  <Button variant="ghost" size="sm" className="gap-1 text-xs hover:bg-primary/10 hover:text-primary transition-colors">
                    Показать все
                    <ArrowUpRight className="w-3 h-3" />
                  </Button>
                </Link>
              </CardHeader>
              <CardContent className="p-0">
                <div className="divide-y divide-border">
                  {auditLoading ? (
                    Array(5).fill(0).map((_, i) => (
                      <div key={i} className="p-4 flex items-center gap-4">
                        <Skeleton className="h-10 w-10 rounded-full" />
                        <div className="space-y-2 flex-1">
                          <Skeleton className="h-4 w-1/2" />
                          <Skeleton className="h-3 w-1/4" />
                        </div>
                      </div>
                    ))
                  ) : auditLogs && auditLogs.length > 0 ? (
                    auditLogs.map((log) => (
                      <div key={log.id} className="p-4 flex items-start gap-4 hover:bg-muted/30 transition-colors group">
                        <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center shrink-0 text-muted-foreground group-hover:bg-primary/10 group-hover:text-primary transition-colors">
                          <Activity className="w-5 h-5" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between mb-0.5">
                            <span className="text-sm font-semibold truncate">
                              {log.action}
                            </span>
                            <span className="text-[10px] text-muted-foreground flex items-center gap-1 font-medium bg-muted px-1.5 py-0.5 rounded">
                              <Clock className="w-3 h-3" />
                              {format(new Date(log.created_at), "HH:mm")}
                            </span>
                          </div>
                          <p className="text-xs text-muted-foreground truncate">
                            {log.entity} &bull; {log.description.slice(0, 50)}{log.description.length > 50 ? '...' : ''}
                          </p>
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className="p-8 text-center text-muted-foreground italic text-sm">
                      Нет недавней активности
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
            
            <Card className="border-border shadow-sm">
              <CardHeader>
                <CardTitle className="text-xl">Быстрые действия</CardTitle>
                <CardDescription>Популярные инструменты администрирования</CardDescription>
              </CardHeader>
              <CardContent className="grid grid-cols-2 gap-3">
                <Link href="/users">
                  <Button variant="outline" className="w-full justify-start gap-2 h-auto py-3 bg-muted/30 border-border/50 hover:bg-primary/5 hover:border-primary/30 group">
                    <div className="w-8 h-8 rounded-lg bg-background flex items-center justify-center border border-border group-hover:bg-primary/10 transition-colors">
                      <Users className="w-4 h-4 text-primary" />
                    </div>
                    <div className="text-left">
                      <p className="text-sm font-medium">Пользователи</p>
                      <p className="text-[10px] text-muted-foreground">Управление доступом</p>
                    </div>
                  </Button>
                </Link>
                <Link href="/profile">
                  <Button variant="outline" className="w-full justify-start gap-2 h-auto py-3 bg-muted/30 border-border/50 hover:bg-primary/5 hover:border-primary/30 group">
                    <div className="w-8 h-8 rounded-lg bg-background flex items-center justify-center border border-border group-hover:bg-primary/10 transition-colors">
                      <Shield className="w-4 h-4 text-emerald-500" />
                    </div>
                    <div className="text-left">
                      <p className="text-sm font-medium">Безопасность</p>
                      <p className="text-[10px] text-muted-foreground">Личные настройки</p>
                    </div>
                  </Button>
                </Link>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: number;
  icon: LucideIcon;
  trend?: string;
  color?: string;
}

function StatCard({ title, value, icon: Icon, trend, color = "text-foreground" }: StatCardProps) {
  return (
    <Card className="border-border shadow-sm overflow-hidden group hover:border-primary/50 transition-colors">
      <CardContent className="p-4 md:p-5 flex flex-col justify-between h-full relative">
        <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity transform group-hover:scale-110 duration-500">
          <Icon className="w-12 h-12" />
        </div>
        <div className="flex items-center gap-2 text-muted-foreground mb-4">
          <Icon className="w-4 h-4" />
          <span className="text-xs font-medium tracking-wide uppercase">{title}</span>
        </div>
        <div className="flex items-end justify-between z-10">
          <span className={`text-3xl font-bold tracking-tight ${color}`}>{value}</span>
          {trend && <span className="text-xs font-medium text-emerald-500 bg-emerald-500/10 px-1.5 py-0.5 rounded">{trend}</span>}
        </div>
      </CardContent>
    </Card>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-8">
      <div>
        <Skeleton className="h-10 w-48 mb-2" />
        <Skeleton className="h-5 w-64" />
      </div>
      <div className="space-y-6">
        <div className="space-y-4">
          <Skeleton className="h-7 w-40" />
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {[1, 2, 3, 4].map(i => <Skeleton key={i} className="h-28 rounded-xl" />)}
          </div>
        </div>
        <Skeleton className="h-64 rounded-xl" />
      </div>
    </div>
  );
}