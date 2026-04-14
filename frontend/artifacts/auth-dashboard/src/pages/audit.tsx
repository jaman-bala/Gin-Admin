import React, { useState } from "react";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import { 
  ListChecks, 
  Search, 
  RefreshCw, 
  Eye, 
  Shield, 
  User as UserIcon, 
  Globe, 
  Activity,
  CheckCircle2,
  XCircle,
  Clock
} from "lucide-react";
import { useTranslation } from "@/hooks/useTranslation";
import { useGetAuditLogs } from "@workspace/api-client-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PaginationControl } from "@/components/ui/pagination-control";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

const AuditPage = () => {
  const { t } = useTranslation();
  const [searchTerm, setSearchTerm] = useState("");
  const [page, setPage] = useState(1);
  const [selectedLog, setSelectedLog] = useState<any>(null);
  const { data, isLoading, refetch, isRefetching } = useGetAuditLogs({ page, limit: 10 });

  const logs = data?.logs || [];
  const total = data?.total || 0;

  const filteredLogs = logs.filter(log => 
    log.action.toLowerCase().includes(searchTerm.toLowerCase()) ||
    log.entity.toLowerCase().includes(searchTerm.toLowerCase()) ||
    log.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t.audit_page.title}</h1>
          <p className="text-muted-foreground mt-1">{t.audit_page.description}</p>
        </div>
        <Button variant="outline" onClick={() => refetch()} disabled={isRefetching} className="gap-2">
          <RefreshCw className={cn("w-4 h-4", isRefetching && "animate-spin")} />
          {isRefetching ? "Обновление..." : "Обновить"}
        </Button>
      </div>

      <Card className="border-border/50 shadow-sm overflow-hidden">
        <CardContent className="p-0">
          <div className="p-4 border-b border-border bg-muted/30">
            <div className="relative max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder={t.audit_page.search_placeholder}
                className="pl-9 h-10"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
          </div>

          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/20">
                  <TableHead className="w-[180px]">{t.audit_page.table_date}</TableHead>
                  <TableHead>{t.audit_page.table_user}</TableHead>
                  <TableHead>{t.audit_page.table_action}</TableHead>
                  <TableHead>{t.audit_page.table_entity}</TableHead>
                  <TableHead>{t.audit_page.table_ip}</TableHead>
                  <TableHead className="w-[100px] text-right">{t.common.actions}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  Array(5).fill(0).map((_, i) => (
                    <TableRow key={i}>
                      {Array(6).fill(0).map((_, j) => (
                        <TableCell key={j}><div className="h-4 bg-muted animate-pulse rounded" /></TableCell>
                      ))}
                    </TableRow>
                  ))
                ) : filteredLogs?.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                      <div className="flex flex-col items-center justify-center gap-2">
                        <Activity className="w-8 h-8 opacity-20" />
                        <p>{t.audit_page.no_logs}</p>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredLogs?.map((log) => (
                    <TableRow key={log.id} className="group hover:bg-muted/30 transition-colors">
                      <TableCell className="text-sm font-medium">
                        <div className="flex flex-col">
                          <span>{format(new Date(log.created_at), "d MMM yyyy", { locale: ru })}</span>
                          <span className="text-xs text-muted-foreground flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                            {format(new Date(log.created_at), "HH:mm:ss")}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary">
                            <UserIcon className="w-4 h-4" />
                          </div>
                          <span className="text-sm font-mono truncate max-w-[120px]">
                            {log.user_id.slice(0, 8)}...
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={cn(
                          "font-semibold border-border bg-background",
                          log.action.includes("DELETE") ? "text-destructive border-destructive/20" :
                          log.action.includes("CREATE") ? "text-emerald-600 border-emerald-200" :
                          "text-blue-600 border-blue-200"
                        )}>
                          {log.action}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1.5">
                          <Shield className="w-3 h-3 text-muted-foreground" />
                          <span className="text-sm">{log.entity}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-mono">
                          <Globe className="w-3 h-3" />
                          {log.client_ip}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <Button 
                          variant="ghost" 
                          size="icon" 
                          className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                          onClick={() => setSelectedLog(log)}
                        >
                          <Eye className="w-4 h-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
          <div className="p-4 border-t border-border bg-muted/10 flex flex-col sm:flex-row items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground order-2 sm:order-1">
              Показано <span className="font-medium text-foreground">{(page - 1) * 10 + 1}</span> - <span className="font-medium text-foreground">{Math.min(page * 10, total)}</span> из <span className="font-medium text-foreground">{total}</span> записей
            </span>
            <div className="order-1 sm:order-2">
              <PaginationControl
                currentPage={page}
                totalCount={total}
                pageSize={10}
                onPageChange={setPage}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog open={!!selectedLog} onOpenChange={() => setSelectedLog(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <ListChecks className="w-5 h-5 text-primary" />
              {t.audit_page.details}
            </DialogTitle>
            <DialogDescription>
              Полная информация о системном событии #{selectedLog?.id}
            </DialogDescription>
          </DialogHeader>
          
          {selectedLog && (
            <div className="grid grid-cols-2 gap-6 py-4">
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-semibold text-muted-foreground mb-1">Действие</h4>
                  <p className="text-sm font-medium bg-muted p-2 rounded border border-border">{selectedLog.action}</p>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-muted-foreground mb-1">Сущность</h4>
                  <p className="text-sm font-medium bg-muted p-2 rounded border border-border">{selectedLog.entity} ({selectedLog.entity_id})</p>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-muted-foreground mb-1">Пользователь</h4>
                  <p className="text-sm font-mono bg-muted p-2 rounded border border-border">{selectedLog.user_id}</p>
                </div>
              </div>
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm font-semibold text-muted-foreground mb-1">Клиент</h4>
                  <div className="text-sm bg-muted p-2 rounded border border-border space-y-1">
                    <p className="font-mono flex items-center gap-2"><Globe className="w-3 h-3" /> {selectedLog.client_ip}</p>
                    <p className="text-[10px] break-all text-muted-foreground leading-tight">{selectedLog.user_agent}</p>
                  </div>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-muted-foreground mb-1">Статус</h4>
                  <div className="flex items-center gap-2 bg-muted p-2 rounded border border-border">
                    {selectedLog.status === 200 ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                    ) : (
                      <XCircle className="w-4 h-4 text-destructive" />
                    )}
                    <span className="text-sm font-medium">HTTP {selectedLog.status}</span>
                  </div>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-muted-foreground mb-1">Дата</h4>
                  <p className="text-sm bg-muted p-2 rounded border border-border font-medium">
                    {format(new Date(selectedLog.created_at), "PPPP pppp", { locale: ru })}
                  </p>
                </div>
              </div>
              <div className="col-span-2">
                <h4 className="text-sm font-semibold text-muted-foreground mb-1">Описание / Данные</h4>
                <pre className="text-[11px] bg-slate-950 text-slate-100 p-4 rounded-lg overflow-x-auto border border-border shadow-inner font-mono leading-relaxed">
                  {selectedLog.description}
                </pre>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default AuditPage;
