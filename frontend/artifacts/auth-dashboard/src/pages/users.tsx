import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { 
  useListUsers, 
  getListUsersQueryKey,
  useDeleteUser,
  useUpdateUser,
  User,
  getGetUserStatsQueryKey
} from "@workspace/api-client-react";
import { useAuthStore } from "@/store/authStore";
import { useTranslation } from "@/hooks/useTranslation";
import { Redirect } from "wouter";

import { useToast } from "@/hooks/use-toast";
import { Button } from "@/components/ui/button";
import { 
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Plus, ShieldAlert, Ban, CheckCircle2, Trash2, LogIn } from "lucide-react";
import { cn } from "@/lib/utils";

// Modular Components
import { UserTable } from "@/components/users/UserTable";
import { UserFilters } from "@/components/users/UserFilters";
import { CreateUserDialog, EditUserDialog } from "@/components/users/UserDialogs";
import { PaginationControl } from "@/components/ui/pagination-control";

export default function UsersPage() {
  const { t } = useTranslation();
  const { user: currentUser } = useAuthStore();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [activeFilter, setActiveFilter] = useState<string>("all");
  
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [isBulkDeleteOpen, setIsBulkDeleteOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  // Guard: Admin only (admin or superuser)
  const isAdmin = currentUser?.role === "admin" || currentUser?.role === "superuser";
  if (currentUser && !isAdmin) {
    return <Redirect to="/dashboard" />;
  }

  // Fetch users
  const is_active_param = activeFilter === "all" ? undefined : activeFilter === "active";
  const { data, isLoading, refetch, isRefetching } = useListUsers(
    { page, limit: 10, search: search || undefined, is_active: is_active_param },
    { 
      query: { 
        queryKey: getListUsersQueryKey({ page, limit: 10, search: search || undefined, is_active: is_active_param }),
        placeholderData: (previousData: any) => previousData
      } 
    }
  );

  const deleteMutation = useDeleteUser({
    mutation: {
      onSuccess: () => {
        toast({ title: t.common.success, description: "Пользователь удален" });
        queryClient.invalidateQueries({ queryKey: getListUsersQueryKey() });
        queryClient.invalidateQueries({ queryKey: getGetUserStatsQueryKey() });
        setIsDeleteOpen(false);
        setIsBulkDeleteOpen(false);
        setSelectedIds([]);
      },
      onError: (err: any) => {
        toast({ 
          title: t.common.error, 
          description: err.response?.data?.message || t.common.error,
          variant: "destructive" 
        });
      }
    }
  });

  const updateMutation = useUpdateUser({
    mutation: {
      onSuccess: () => {
        toast({ title: t.common.success, description: t.common.changes_saved });
        queryClient.invalidateQueries({ queryKey: getListUsersQueryKey() });
        queryClient.invalidateQueries({ queryKey: getGetUserStatsQueryKey() });
      },
      onError: (err: any) => {
        toast({ 
          title: t.common.error, 
          description: err.response?.data?.message || t.common.error,
          variant: "destructive" 
        });
      }
    }
  });

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setSearch(searchInput);
    setPage(1);
  };

  const confirmDelete = (user: User) => {
    setSelectedUser(user);
    setIsDeleteOpen(true);
  };

  const openEdit = (user: User) => {
    setSelectedUser(user);
    setIsEditOpen(true);
  };

  const handleSelect = (id: string, selected: boolean) => {
    if (selected) {
      setSelectedIds((prev) => [...prev, id]);
    } else {
      setSelectedIds((prev) => prev.filter((i) => i !== id));
    }
  };

  const handleSelectAll = (selected: boolean) => {
    if (selected) {
      setSelectedIds(users.map((u) => u.id));
    } else {
      setSelectedIds([]);
    }
  };

  const handleToggleStatus = (user: User) => {
    updateMutation.mutate({
      id: user.id,
      data: { is_active: !user.is_active }
    });
  };


  const handleBulkDelete = () => {
    setIsBulkDeleteOpen(true);
  };

  const confirmBulkDelete = () => {
    selectedIds.forEach((id) => {
      if (id !== currentUser?.id) {
        deleteMutation.mutate({ id });
      }
    });
  };

  const handleBulkActivate = () => {
    selectedIds.forEach((id) => {
      updateMutation.mutate({ id, data: { is_active: true } });
    });
  };

  const handleBulkDeactivate = () => {
    selectedIds.forEach((id) => {
      if (id !== currentUser?.id) {
        updateMutation.mutate({ id, data: { is_active: false } });
      }
    });
  };

  const users = data?.users || [];
  const totalPages = data ? Math.ceil(data.total / data.limit) : 1;
  const selectedUsers = users.filter((u) => selectedIds.includes(u.id));
  const canBulkDelete = selectedUsers.some((u) => u.id !== currentUser?.id);

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t.users_page.title}</h1>
          <p className="text-muted-foreground mt-1">{t.users_page.description}</p>
        </div>
        <Button onClick={() => setIsCreateOpen(true)} className="gap-2 shadow-sm">
          <Plus className="w-4 h-4" />
          {t.users_page.create_user}
        </Button>
      </div>

      <div className="bg-card border border-border rounded-xl shadow-sm overflow-hidden flex flex-col">
        <UserFilters 
          searchInput={searchInput}
          setSearchInput={setSearchInput}
          search={search}
          onSearchSubmit={handleSearchSubmit}
          activeFilter={activeFilter}
          onFilterChange={(val) => { setActiveFilter(val); setPage(1); }}
          onRefresh={refetch}
          isRefetching={isRefetching}
        />

        {/* Bulk Actions Bar */}
        {selectedIds.length > 0 && (
          <div className="px-4 py-3 border-b border-border bg-muted/30 flex items-center justify-between animate-in slide-in-from-top-2">
            <span className="text-sm font-medium">
              Выбрано <span className="font-bold">{selectedIds.length}</span> пользователей
            </span>
            <div className="flex items-center gap-2">
              <Button 
                variant="outline" 
                size="sm" 
                onClick={handleBulkActivate}
                disabled={updateMutation.isPending}
                className="gap-1.5 text-emerald-600 border-emerald-200 hover:bg-emerald-50"
              >
                <CheckCircle2 className="w-4 h-4" />
                Активировать
              </Button>
              <Button 
                variant="outline" 
                size="sm" 
                onClick={handleBulkDeactivate}
                disabled={updateMutation.isPending}
                className="gap-1.5 text-amber-600 border-amber-200 hover:bg-amber-50"
              >
                <Ban className="w-4 h-4" />
                Деактивировать
              </Button>
              <Button 
                variant="outline" 
                size="sm" 
                onClick={handleBulkDelete}
                disabled={deleteMutation.isPending || !canBulkDelete}
                className="gap-1.5 text-destructive border-destructive/20 hover:bg-destructive/10"
              >
                <Trash2 className="w-4 h-4" />
                Удалить
              </Button>
            </div>
          </div>
        )}

        <div className="overflow-x-auto">
          <UserTable 
            users={users}
            isLoading={isLoading}
            currentUserId={currentUser?.id}
            selectedIds={selectedIds}
            onSelect={handleSelect}
            onSelectAll={handleSelectAll}
            onEdit={openEdit}
            onDelete={confirmDelete}
            onToggleStatus={handleToggleStatus}
          />
        </div>

        {/* Pagination */}
        <div className="p-4 border-t border-border bg-muted/10 flex flex-col sm:flex-row items-center justify-between gap-4">
          <span className="text-sm text-muted-foreground order-2 sm:order-1">
            Показано <span className="font-medium text-foreground">{(page - 1) * 10 + 1}</span> - <span className="font-medium text-foreground">{Math.min(page * 10, data?.total || 0)}</span> из <span className="font-medium text-foreground">{data?.total || 0}</span> пользователей
          </span>
          <div className="order-1 sm:order-2">
            <PaginationControl
              currentPage={page}
              totalCount={data?.total || 0}
              pageSize={10}
              onPageChange={setPage}
            />
          </div>
        </div>
      </div>

      <CreateUserDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} />
      
      {selectedUser && (
        <EditUserDialog 
          user={selectedUser} 
          open={isEditOpen} 
          onOpenChange={(open) => {
            setIsEditOpen(open);
            if (!open) setTimeout(() => setSelectedUser(null), 200);
          }} 
        />
      )}

      {/* Single Delete Dialog */}
      <AlertDialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <ShieldAlert className="w-5 h-5 text-destructive" />
              {t.common.confirm_action}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t.users_page.delete_description} <strong>{selectedUser?.first_name} {selectedUser?.last_name}</strong>. {t.common.confirm_delete}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>{t.common.cancel}</AlertDialogCancel>
            <AlertDialogAction 
              onClick={(e) => { e.preventDefault(); if (selectedUser) deleteMutation.mutate({ id: selectedUser.id }); }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? t.common.loading : t.common.delete}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Bulk Delete Dialog */}
      <AlertDialog open={isBulkDeleteOpen} onOpenChange={setIsBulkDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <ShieldAlert className="w-5 h-5 text-destructive" />
              {t.common.confirm_action}
            </AlertDialogTitle>
            <AlertDialogDescription>
              Вы собираетесь удалить <strong>{selectedIds.length}</strong> пользователей. 
              Это действие нельзя отменить. {t.common.confirm_delete}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>{t.common.cancel}</AlertDialogCancel>
            <AlertDialogAction 
              onClick={(e) => { e.preventDefault(); confirmBulkDelete(); }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? t.common.loading : t.common.delete}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
