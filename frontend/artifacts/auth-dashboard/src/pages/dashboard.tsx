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
import { format, subDays } from "date-fns";
import { ru } from "date-fns/locale";
import { useAuthStore } from "@/store/authStore";
import { useTranslation } from "@/hooks/useTranslation";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Activity,
  Users,
  Shield,
  UserPlus,
  LucideIcon,
  ListChecks,
  ArrowUpRight,
  Clock,
  User,
  FileText,
  ShieldCheck
} from "lucide-react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Cell
} from "recharts";

// Generate activity chart data from audit logs
function buildActivityData(logs: any[]) {
  const days: Record<string, number> = {};
  for (let i = 6; i >= 0; i--) {
    const d = format(subDays(new Date(), i), "dd.MM");
    days[d] = 0;
  }
  logs.forEach((log) => {
    const d = format(new Date(log.created_at), "dd.MM");
    if (d in days) days[d]++;
  });
  return Object.entries(days).map(([date, count]) => ({ date, count }));
}

function buildRoleData(stats: any) {
  if (!stats) return [];
  return [
    { name: "Пользователи", value: Math.max(0, stats.total - stats.admins), color: "#6366f1" },
    { name: "Администраторы", value: stats.admins || 0, color: "#22c55e" },
  ];
}

export default function Dashboard() {
  const { t } = useTranslation();
  const { user: storeUser, setUser } = useAuthStore();

  const { data: me, isLoading: meLoading } = useGetMe({
    query: { queryKey: getGetMeQueryKey(), refetchOnWindowFocus: false }
  });

  useEffect(() => { if (me) setUser(me); }, [me, setUser]);

  const currentUser = me || storeUser;
  const isAdmin = currentUser?.role === "admin" || currentUser?.role === "superuser";

  const { data: stats, isLoading: statsLoading } = useGetUserStats({
    query: { queryKey: getGetUserStatsQueryKey(), enabled: !!currentUser && isAdmin }
  });

  const { data: auditLogsRaw, isLoading: auditLoading } = useGetAuditLogs({
    query: { queryKey: getGetAuditLogsQueryKey(), enabled: !!currentUser && isAdmin }
  });

  const auditLogs = auditLogsRaw?.logs?.slice(0, 6) ?? [];
  const activityData = buildActivityData(auditLogsRaw?.logs ?? []);
  const roleData = buildRoleData(stats);

  if (meLoading && !currentUser) return <DashboardSkeleton />;

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t.common.dashboard}</h1>
        <p className="text-muted-foreground text-sm mt-0.5">{t.dashboard.system_overview}</p>
      </div>

      {isAdmin && (
        <>
          {/* Stat Cards */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {statsLoading ? (
              [1, 2, 3, 4].map(i => <Skeleton key={i} className="h-28 rounded-xl" />)
            ) : (
              <>
                <StatCard
                  title={t.users_page.total_users}
                  value={stats?.total ?? 0}
                  icon={Users}
                  accent="text-indigo-400"
                  bg="bg-indigo-500/10"
                />
                <StatCard
                  title={t.common.active}
                  value={stats?.active ?? 0}
                  icon={Activity}
                  accent="text-emerald-400"
                  bg="bg-emerald-500/10"
                />
                <StatCard
                  title={t.dashboard.admins}
                  value={stats?.admins ?? 0}
                  icon={Shield}
                  accent="text-primary"
                  bg="bg-primary/10"
                />
                <StatCard
                  title={t.dashboard.new_month}
                  value={stats?.new_this_month ?? 0}
                  icon={UserPlus}
                  accent="text-amber-400"
                  bg="bg-amber-500/10"
                />
              </>
            )}
          </div>

          {/* Charts Row */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {/* Activity Area Chart */}
            <Card className="lg:col-span-2 border-border shadow-sm">
              <CardHeader className="pb-2">
                <CardTitle className="text-base font-semibold">Активность за неделю</CardTitle>
                <CardDescription className="text-xs">Количество событий по дням</CardDescription>
              </CardHeader>
              <CardContent>
                {auditLoading ? (
                  <Skeleton className="h-48 w-full rounded-lg" />
                ) : (
                  <ResponsiveContainer width="100%" height={180}>
                    <AreaChart data={activityData} margin={{ top: 4, right: 8, left: -24, bottom: 0 }}>
                      <defs>
                        <linearGradient id="activityGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#6366f1" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" vertical={false} />
                      <XAxis dataKey="date" tick={{ fontSize: 11, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
                      <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
                      <Tooltip
                        contentStyle={{ background: "hsl(var(--card))", border: "1px solid hsl(var(--border))", borderRadius: 8, fontSize: 12 }}
                        labelStyle={{ color: "hsl(var(--foreground))", fontWeight: 600 }}
                        cursor={{ stroke: "hsl(var(--border))" }}
                        formatter={(v: number) => [v, "События"]}
                      />
                      <Area type="monotone" dataKey="count" stroke="#6366f1" strokeWidth={2} fill="url(#activityGrad)" dot={{ fill: "#6366f1", r: 3 }} activeDot={{ r: 5 }} />
                    </AreaChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>

            {/* Role Distribution Bar Chart */}
            <Card className="border-border shadow-sm">
              <CardHeader className="pb-2">
                <CardTitle className="text-base font-semibold">Роли пользователей</CardTitle>
                <CardDescription className="text-xs">Распределение по ролям</CardDescription>
              </CardHeader>
              <CardContent>
                {statsLoading ? (
                  <Skeleton className="h-48 w-full rounded-lg" />
                ) : (
                  <div className="space-y-4">
                    <ResponsiveContainer width="100%" height={120}>
                      <BarChart data={roleData} margin={{ top: 4, right: 8, left: -24, bottom: 0 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" vertical={false} />
                        <XAxis dataKey="name" tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
                        <YAxis allowDecimals={false} tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
                        <Tooltip
                          contentStyle={{ background: "hsl(var(--card))", border: "1px solid hsl(var(--border))", borderRadius: 8, fontSize: 12 }}
                          cursor={{ fill: "hsl(var(--muted))" }}
                          formatter={(v: number) => [v, "чел."]}
                        />
                        <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                          {roleData.map((entry, i) => <Cell key={i} fill={entry.color} />)}
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                    <div className="space-y-2">
                      {roleData.map((r) => (
                        <div key={r.name} className="flex items-center justify-between text-sm">
                          <div className="flex items-center gap-2">
                            <div className="w-2.5 h-2.5 rounded-full" style={{ background: r.color }} />
                            <span className="text-muted-foreground text-xs">{r.name}</span>
                          </div>
                          <span className="font-semibold tabular-nums">{r.value}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Bottom Row */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {/* Recent Activity */}
            <Card className="lg:col-span-2 border-border shadow-sm overflow-hidden">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
                <div>
                  <CardTitle className="text-base font-semibold flex items-center gap-2">
                    <ListChecks className="w-4 h-4 text-primary" />
                    Недавняя активность
                  </CardTitle>
                  <CardDescription className="text-xs">Последние события в системе</CardDescription>
                </div>
                <Link href="/audit">
                  <Button variant="ghost" size="sm" className="gap-1 text-xs h-8 hover:text-primary">
                    Все события
                    <ArrowUpRight className="w-3 h-3" />
                  </Button>
                </Link>
              </CardHeader>
              <CardContent className="p-0">
                <div className="divide-y divide-border">
                  {auditLoading ? (
                    Array(4).fill(0).map((_, i) => (
                      <div key={i} className="px-6 py-3 flex items-center gap-3">
                        <Skeleton className="h-8 w-8 rounded-full shrink-0" />
                        <div className="space-y-1.5 flex-1"><Skeleton className="h-3.5 w-1/2" /><Skeleton className="h-3 w-1/4" /></div>
                      </div>
                    ))
                  ) : auditLogs.length > 0 ? (
                    auditLogs.map((log: any) => (
                      <div key={log.id} className="px-6 py-3 flex items-center gap-3 hover:bg-muted/30 transition-colors">
                        <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                          <Activity className="w-4 h-4 text-primary" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between">
                            <span className="text-sm font-medium truncate">{log.action}</span>
                            <span className="text-[10px] text-muted-foreground flex items-center gap-1 shrink-0 ml-2">
                              <Clock className="w-3 h-3" />
                              {format(new Date(log.created_at), "HH:mm", { locale: ru })}
                            </span>
                          </div>
                          <p className="text-xs text-muted-foreground truncate">{log.entity}</p>
                        </div>
                        <Badge variant="outline" className={`text-[10px] shrink-0 ${log.status === 200 ? "border-emerald-500/30 text-emerald-500" : "border-destructive/30 text-destructive"}`}>
                          {log.status}
                        </Badge>
                      </div>
                    ))
                  ) : (
                    <div className="px-6 py-10 text-center">
                      <FileText className="w-8 h-8 text-muted-foreground/30 mx-auto mb-2" />
                      <p className="text-sm text-muted-foreground">Нет активности</p>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Quick Actions */}
            <Card className="border-border shadow-sm">
              <CardHeader className="pb-3">
                <CardTitle className="text-base font-semibold">Быстрые действия</CardTitle>
                <CardDescription className="text-xs">Часто используемые разделы</CardDescription>
              </CardHeader>
              <CardContent className="space-y-2">
                {[
                  { href: "/users", icon: Users, label: "Пользователи", desc: "Управление доступом", color: "text-indigo-400 bg-indigo-500/10" },
                  { href: "/users", icon: UserPlus, label: "Добавить пользователя", desc: "Создать новую запись", color: "text-emerald-400 bg-emerald-500/10" },
                  { href: "/audit", icon: ShieldCheck, label: "Журнал аудита", desc: "События системы", color: "text-amber-400 bg-amber-500/10" },
                  { href: "/profile", icon: User, label: "Мой профиль", desc: "Настройки аккаунта", color: "text-primary bg-primary/10" },
                ].map((item) => (
                  <Link key={item.label} href={item.href}>
                    <div className="flex items-center gap-3 p-3 rounded-lg border border-border/50 hover:border-border hover:bg-muted/30 transition-all cursor-pointer group">
                      <div className={`w-8 h-8 rounded-lg flex items-center justify-center shrink-0 ${item.color}`}>
                        <item.icon className="w-4 h-4" />
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-medium leading-tight group-hover:text-primary transition-colors">{item.label}</p>
                        <p className="text-[11px] text-muted-foreground">{item.desc}</p>
                      </div>
                      <ArrowUpRight className="w-3.5 h-3.5 text-muted-foreground/40 ml-auto shrink-0 group-hover:text-primary transition-colors" />
                    </div>
                  </Link>
                ))}
              </CardContent>
            </Card>
          </div>
        </>
      )}

      {/* Non-admin welcome */}
      {!isAdmin && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card className="border-border shadow-sm md:col-span-2">
            <CardHeader>
              <CardTitle>{t.dashboard.welcome_title}</CardTitle>
              <CardDescription>{t.dashboard.welcome_description}</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">{t.dashboard.platform_info} {t.dashboard.user_access}</p>
              <div className="mt-4 p-4 bg-muted/50 rounded-lg border border-border">
                <h4 className="font-medium text-sm mb-1">{t.dashboard.need_help}</h4>
                <p className="text-muted-foreground text-sm">{t.dashboard.contact_admin}</p>
              </div>
            </CardContent>
          </Card>
          <Link href="/profile">
            <Card className="border-border shadow-sm hover:border-primary/30 transition-colors cursor-pointer group">
              <CardContent className="p-5 flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                  <User className="w-5 h-5 text-primary" />
                </div>
                <div>
                  <p className="font-medium group-hover:text-primary transition-colors">Мой профиль</p>
                  <p className="text-xs text-muted-foreground">Личные данные и безопасность</p>
                </div>
                <ArrowUpRight className="w-4 h-4 text-muted-foreground/40 ml-auto group-hover:text-primary transition-colors" />
              </CardContent>
            </Card>
          </Link>
        </div>
      )}
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: number;
  icon: LucideIcon;
  accent: string;
  bg: string;
}

function StatCard({ title, value, icon: Icon, accent, bg }: StatCardProps) {
  return (
    <Card className="border-border shadow-sm overflow-hidden hover:shadow-md transition-shadow">
      <CardContent className="p-5">
        <div className="flex items-start justify-between mb-4">
          <div className={`w-9 h-9 rounded-lg ${bg} flex items-center justify-center`}>
            <Icon className={`w-5 h-5 ${accent}`} />
          </div>
        </div>
        <div>
          <p className="text-2xl font-bold tabular-nums">{value.toLocaleString()}</p>
          <p className="text-xs text-muted-foreground mt-0.5 uppercase tracking-wide font-medium">{title}</p>
        </div>
      </CardContent>
    </Card>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <div><Skeleton className="h-8 w-40 mb-1" /><Skeleton className="h-4 w-56" /></div>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">{[1,2,3,4].map(i => <Skeleton key={i} className="h-28 rounded-xl" />)}</div>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4"><Skeleton className="lg:col-span-2 h-56 rounded-xl" /><Skeleton className="h-56 rounded-xl" /></div>
    </div>
  );
}