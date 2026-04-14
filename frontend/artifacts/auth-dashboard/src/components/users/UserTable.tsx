import { format } from "date-fns";
import { ru } from "date-fns/locale";
import {
  MoreHorizontal,
  Edit,
  Trash2,
  ShieldCheck,
  User as UserIcon,
  Ban,
  CheckCircle2,
  Clock,
} from "lucide-react";
import { User } from "@workspace/api-client-react";
import { useTranslation } from "@/hooks/useTranslation";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface UserTableProps {
  users: User[];
  isLoading: boolean;
  currentUserId?: string;
  selectedIds: string[];
  onSelect: (id: string, selected: boolean) => void;
  onSelectAll: (selected: boolean) => void;
  onEdit: (user: User) => void;
  onDelete: (user: User) => void;
  onToggleStatus?: (user: User) => void;
}

export function UserTable({
  users,
  isLoading,
  currentUserId,
  selectedIds,
  onSelect,
  onSelectAll,
  onEdit,
  onDelete,
  onToggleStatus,
}: UserTableProps) {
  const { t } = useTranslation();
  const allSelected = users.length > 0 && users.every((u) => selectedIds.includes(u.id));
  const someSelected = selectedIds.length > 0 && !allSelected;

  if (isLoading) {
    return (
      <Table>
        <TableHeader>
          <TableRow className="bg-muted/30 hover:bg-muted/30">
            <TableHead className="w-12">
              <Checkbox
                checked={allSelected}
                onCheckedChange={(checked) => onSelectAll(!!checked)}
                aria-label="Select all"
              />
            </TableHead>
            <TableHead>{t.auth.first_name} / {t.auth.last_name}</TableHead>
            <TableHead>{t.common.phone}</TableHead>
            <TableHead>{t.common.role}</TableHead>
            <TableHead>{t.common.status}</TableHead>
            <TableHead>{t.users_page.joined}</TableHead>
            <TableHead className="text-right">{t.common.actions}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array(5)
            .fill(0)
            .map((_, i) => (
              <TableRow key={i}>
                <TableCell>
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-muted animate-pulse" />
                    <div className="space-y-2">
                      <div className="h-4 w-32 bg-muted animate-pulse rounded" />
                      <div className="h-3 w-24 bg-muted animate-pulse rounded" />
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="h-4 w-24 bg-muted animate-pulse rounded" />
                </TableCell>
                <TableCell>
                  <div className="h-5 w-16 bg-muted animate-pulse rounded-full" />
                </TableCell>
                <TableCell>
                  <div className="h-5 w-16 bg-muted animate-pulse rounded-full" />
                </TableCell>
                <TableCell>
                  <div className="h-4 w-24 bg-muted animate-pulse rounded" />
                </TableCell>
                <TableCell className="text-right">
                  <div className="h-8 w-8 ml-auto bg-muted animate-pulse rounded" />
                </TableCell>
              </TableRow>
            ))}
        </TableBody>
      </Table>
    );
  }

  if (users.length === 0) {
    return (
      <div className="h-48 flex flex-col items-center justify-center text-muted-foreground">
        <UserIcon className="w-10 h-10 mb-3 text-muted-foreground/50" />
        <p>{t.users_page.description}</p>
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow className="bg-muted/30 hover:bg-muted/30">
          <TableHead>{t.auth.first_name} / {t.auth.last_name}</TableHead>
          <TableHead>{t.common.phone}</TableHead>
          <TableHead>{t.common.role}</TableHead>
          <TableHead>{t.common.status}</TableHead>
          <TableHead>{t.users_page.joined}</TableHead>
          <TableHead className="text-right">{t.common.actions}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {users.map((user) => (
          <TableRow key={user.id} className={cn("group", selectedIds.includes(user.id) && "bg-muted/50")}>
            <TableCell className="w-12">
              <Checkbox
                checked={selectedIds.includes(user.id)}
                onCheckedChange={(checked) => onSelect(user.id, !!checked)}
                aria-label={`Select ${user.first_name}`}
              />
            </TableCell>
            <TableCell>
              <div className="flex items-center gap-3">
                <TooltipProvider delayDuration={100}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="relative">
                        <Avatar className="w-10 h-10 border border-border shadow-sm">
                          <AvatarImage src={user.photo} className="object-cover" />
                          <AvatarFallback className="bg-primary/10 text-primary font-medium">
                            {user.first_name[0]}
                            {user.last_name[0]}
                          </AvatarFallback>
                        </Avatar>
                        <span
                          className={cn(
                            "absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-background",
                            user.is_active ? "bg-emerald-500" : "bg-destructive"
                          )}
                        />
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{user.is_active ? t.common.active : t.common.inactive}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <div className="flex flex-col">
                  <span className="font-medium">
                    {user.first_name} {user.last_name}
                  </span>
                  <span className="text-xs text-muted-foreground font-mono">
                    ID: {user.id.slice(0, 8)}...
                  </span>
                </div>
              </div>
            </TableCell>
            <TableCell className="font-medium text-muted-foreground">
              {user.phone}
            </TableCell>
            <TableCell>
              <Badge
                variant={user.role === "admin" || user.role === "superuser" ? "default" : "secondary"}
                className={cn(
                  "capitalize font-medium",
                  (user.role === "admin" || user.role === "superuser") && "bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 border-amber-500/20",
                  user.role === "user" && "bg-blue-500/10 text-blue-600 hover:bg-blue-500/20 border-blue-500/20"
                )}
              >
                {user.role === "superuser" ? (
                  <ShieldCheck className="w-3 h-3 mr-1 text-primary" />
                ) : user.role === "admin" ? (
                  <ShieldCheck className="w-3 h-3 mr-1" />
                ) : (
                  <UserIcon className="w-3 h-3 mr-1" />
                )}
                {user.role === "superuser" ? t.common.superuser : user.role === "admin" ? t.common.admin : t.common.user}
              </Badge>
            </TableCell>
            <TableCell>
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    "relative inline-flex rounded-full h-2.5 w-2.5",
                    user.is_active ? "bg-emerald-500" : "bg-destructive"
                  )}
                >
                  {user.is_active && (
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                  )}
                </span>
                <span className={cn("text-sm font-medium", user.is_active ? "text-emerald-600" : "text-destructive")}>
                  {user.is_active ? t.common.active : t.common.inactive}
                </span>
              </div>
            </TableCell>
            <TableCell className="text-sm text-muted-foreground">
              <div className="flex flex-col gap-0.5">
                <span>{format(new Date(user.created_at), "d MMM yyyy", { locale: ru })}</span>
                <span className="text-xs text-muted-foreground/70 flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  {format(new Date(user.updated_at), "HH:mm", { locale: ru })}
                </span>
              </div>
            </TableCell>
            <TableCell className="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="opacity-0 group-hover:opacity-100 transition-opacity h-8 w-8 focus:opacity-100"
                  >
                    <MoreHorizontal className="w-4 h-4" />
                    <span className="sr-only">Open menu</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-[200px]">
                  <DropdownMenuLabel>{t.common.actions}</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => onEdit(user)}
                    className="cursor-pointer"
                  >
                    <Edit className="w-4 h-4 mr-2 text-muted-foreground" />
                    {t.common.edit}
                  </DropdownMenuItem>
                  {onToggleStatus && (
                    <DropdownMenuItem
                      onClick={() => onToggleStatus(user)}
                      disabled={user.id === currentUserId}
                      className="cursor-pointer"
                    >
                      {user.is_active ? (
                        <>
                          <Ban className="w-4 h-4 mr-2 text-amber-500" />
                          {t.common.deactivate}
                        </>
                      ) : (
                        <>
                          <CheckCircle2 className="w-4 h-4 mr-2 text-emerald-500" />
                          {t.common.activate}
                        </>
                      )}
                    </DropdownMenuItem>
                  )}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => onDelete(user)}
                    disabled={user.id === currentUserId}
                    className="text-destructive focus:text-destructive focus:bg-destructive/10 cursor-pointer"
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    {t.common.delete}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
