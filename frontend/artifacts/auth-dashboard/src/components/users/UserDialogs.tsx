import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useQueryClient } from "@tanstack/react-query";
import {
  useCreateUser,
  useUpdateUser,
  getListUsersQueryKey,
  getGetUserStatsQueryKey,
  User,
  useGetAuditLogs,
  AuditLog,
} from "@workspace/api-client-react";
import { useTranslation } from "@/hooks/useTranslation";
import { translations } from "@/lib/translations";
import { useToast } from "@/hooks/use-toast";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScrollArea } from "@/components/ui/scroll-area";
import { User as UserIcon, Upload, History, Shield, UserCircle, FileEdit, Trash2, LogIn } from "lucide-react";
import { format } from "date-fns";
import { ru } from "date-fns/locale";
import { cn } from "@/lib/utils";

// ----------------------------------------------------------------------
// Create User Dialog
// ----------------------------------------------------------------------

const createUserSchema = z.object({
  first_name: z.string().min(1, translations.common.required),
  last_name: z.string().min(1, translations.common.required),
  middle_name: z.string().optional(),
  phone: z.string().min(5, translations.common.required),
  password: z.string().min(6, translations.common.min_chars.replace("{count}", "6")),
  role: z.enum(["admin", "user"]),
  telegram: z.string().optional(),
});

type CreateUserValues = z.infer<typeof createUserSchema>;

export function CreateUserDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const createMutation = useCreateUser();
  const [photoBlob, setPhotoBlob] = useState<Blob | null>(null);

  const form = useForm<CreateUserValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: {
      first_name: "",
      last_name: "",
      middle_name: "",
      phone: "",
      password: "",
      role: "user",
      telegram: "",
    },
  });

  const onSubmit = (values: CreateUserValues) => {
    const dataToSubmit = {
      ...values,
      telegram: values.telegram || undefined,
    };
    createMutation.mutate(
      { data: dataToSubmit },
      {
        onSuccess: () => {
          toast({ title: t.common.success, description: t.common.user_created });
          queryClient.invalidateQueries({ queryKey: getListUsersQueryKey() });
          queryClient.invalidateQueries({ queryKey: getGetUserStatsQueryKey() });
          form.reset();
          setPhotoBlob(null);
          onOpenChange(false);
        },
        onError: (err: any) => {
          toast({
            title: t.common.error,
            description: err.response?.data?.message || err.message,
            variant: "destructive",
          });
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{t.users_page.create_user}</DialogTitle>
          <DialogDescription>
            {t.users_page.create_description}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="space-y-4 pt-4"
          >
            <div className="flex items-center gap-4 mb-2">
              <div className="w-16 h-16 rounded-full bg-muted border border-border flex items-center justify-center overflow-hidden flex-shrink-0">
                {photoBlob ? (
                  <img
                    src={URL.createObjectURL(photoBlob)}
                    alt="Preview"
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <UserIcon className="w-8 h-8 text-muted-foreground/50" />
                )}
              </div>
              <div>
                <Label htmlFor="photo-upload" className="cursor-pointer">
                  <div className="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-9 px-3 gap-2">
                    <Upload className="w-4 h-4" />
                    {t.auth.upload_photo}
                  </div>
                </Label>
                <input
                  id="photo-upload"
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={(e) =>
                    e.target.files?.[0] && setPhotoBlob(e.target.files[0])
                  }
                />
                {photoBlob && (
                  <Button
                    variant="link"
                    size="sm"
                    type="button"
                    onClick={() => setPhotoBlob(null)}
                    className="h-8 px-2 text-destructive"
                  >
                    {t.common.delete}
                  </Button>
                )}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="first_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t.auth.first_name}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="last_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t.auth.last_name}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="middle_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t.auth.middle_name} <span className="text-muted-foreground font-normal">({t.common.optional})</span></FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="phone"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t.common.phone}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t.common.password}</FormLabel>
                    <FormControl>
                      <Input type="password" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="role"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t.common.role}</FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      defaultValue={field.value}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder={t.users_page.select_role} />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="user">{t.common.user}</SelectItem>
                        <SelectItem value="admin">{t.common.admin}</SelectItem>
                        <SelectItem value="superuser">{t.common.superuser}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="telegram"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t.auth.telegram}{" "}
                    <span className="text-muted-foreground font-normal">({t.common.optional})</span>
                  </FormLabel>
                  <FormControl>
                    <Input placeholder={t.auth.telegram_placeholder} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter className="pt-4">
              <Button
                variant="outline"
                type="button"
                onClick={() => onOpenChange(false)}
                disabled={createMutation.isPending}
              >
                {t.common.cancel}
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? t.common.loading : t.common.save}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

// ----------------------------------------------------------------------
// Edit User Dialog
// ----------------------------------------------------------------------

const editUserSchema = z.object({
  first_name: z.string().min(1, translations.common.required),
  last_name: z.string().min(1, translations.common.required),
  middle_name: z.string().optional(),
  phone: z.string().min(5, translations.common.required),
  password: z.string().optional().refine(
    (val) => !val || val.length >= 6,
    { message: translations.common.min_chars?.replace("{count}", "6") || "Минимум 6 символов" }
  ),
  role: z.enum(["admin", "user", "superuser"]),
  is_active: z.boolean(),
  telegram: z.string().optional(),
});

type EditUserValues = z.infer<typeof editUserSchema>;

export function EditUserDialog({
  user,
  open,
  onOpenChange,
}: {
  user: User;
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const updateMutation = useUpdateUser();

  const form = useForm<EditUserValues>({
    resolver: zodResolver(editUserSchema),
    defaultValues: {
      first_name: user.first_name,
      last_name: user.last_name,
      middle_name: user.middle_name || "",
      phone: user.phone,
      password: "",
      role: (user.role as "admin" | "user" | "superuser") || "user",
      is_active: user.is_active,
      telegram: user.telegram || "",
    },
  });

  // Update form values if user changes
  useEffect(() => {
    if (user) {
      form.reset({
        first_name: user.first_name,
        last_name: user.last_name,
        middle_name: user.middle_name || "",
        phone: user.phone,
        password: "",
        role: (user.role as "admin" | "user" | "superuser") || "user",
        is_active: user.is_active,
        telegram: user.telegram || "",
      });
    }
  }, [user, form]);

  const onSubmit = (values: EditUserValues) => {
    const dataToSubmit: any = {
      first_name: values.first_name,
      last_name: values.last_name,
      middle_name: values.middle_name,
      phone: values.phone,
      role: values.role,
      is_active: values.is_active,
      telegram: values.telegram || undefined,
    };

    if (values.password) dataToSubmit.password = values.password;

    updateMutation.mutate(
      { id: user.id, data: dataToSubmit },
      {
        onSuccess: () => {
          toast({ title: t.common.success, description: t.common.changes_saved });
          queryClient.invalidateQueries({ queryKey: getListUsersQueryKey() });
          queryClient.invalidateQueries({ queryKey: getGetUserStatsQueryKey() });
          setPhotoBlob(null);
          onOpenChange(false);
        },
        onError: (err: any) => {
          toast({
            title: t.common.error,
            description: err.response?.data?.message || err.message,
            variant: "destructive",
          });
        },
      }
    );
  };

  const { data: auditLogsData, isLoading: isLoadingAudit } = useGetAuditLogs(
    { entity_id: user.id },
    { query: { enabled: open && user.id !== undefined } }
  );

  const auditLogs = auditLogsData?.logs || [];

  const getActionIcon = (action: string) => {
    switch (action) {
      case "LOGIN": return <LogIn className="w-4 h-4 text-blue-500" />;
      case "UPDATE": return <FileEdit className="w-4 h-4 text-amber-500" />;
      case "DELETE": return <Trash2 className="w-4 h-4 text-destructive" />;
      default: return <History className="w-4 h-4 text-muted-foreground" />;
    }
  };

  const getActionColor = (action: string) => {
    switch (action) {
      case "LOGIN": return "bg-blue-500/10 text-blue-600 border-blue-500/20";
      case "UPDATE": return "bg-amber-500/10 text-amber-600 border-amber-500/20";
      case "DELETE": return "bg-destructive/10 text-destructive border-destructive/20";
      default: return "bg-muted text-muted-foreground";
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserCircle className="w-5 h-5 text-primary" />
            {t.users_page.edit_user}
          </DialogTitle>
          <DialogDescription>
            {user.first_name} {user.last_name} · {user.phone}
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="profile" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="profile" className="gap-1.5">
              <UserCircle className="w-4 h-4" />
              Профиль
            </TabsTrigger>
            <TabsTrigger value="audit" className="gap-1.5">
              <History className="w-4 h-4" />
              История действий
            </TabsTrigger>
          </TabsList>

          <TabsContent value="profile" className="mt-4">
            <Form {...form}>
              <form
                onSubmit={form.handleSubmit(onSubmit)}
                className="space-y-4"
              >
                <div className="flex items-center justify-between p-3 rounded-lg border border-border bg-muted/30">
                  <div className="flex flex-col gap-1">
                    <Label>{t.common.status}</Label>
                    <span className="text-xs text-muted-foreground">
                      {t.users_page.platform_access}
                    </span>
                  </div>
                  <FormField
                    control={form.control}
                    name="is_active"
                    render={({ field }: { field: any }) => (
                      <FormItem className="flex items-center space-x-2 space-y-0">
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                        <FormLabel className="font-normal m-0">
                          {field.value ? t.common.active : t.common.inactive}
                        </FormLabel>
                      </FormItem>
                    )}
                  />
                </div>

                <div className="flex items-center gap-4 py-2">
                  <Avatar className="w-16 h-16 border border-border shadow-sm">
                    {photoBlob ? (
                      <AvatarImage
                        src={URL.createObjectURL(photoBlob)}
                        className="object-cover"
                      />
                    ) : (
                      <AvatarImage src={user.photo} className="object-cover" />
                    )}
                    <AvatarFallback className="bg-muted text-muted-foreground text-lg">
                      {user.first_name[0]}
                      {user.last_name[0]}
                    </AvatarFallback>
                  </Avatar>
                  <div>
                    <Label htmlFor="edit-photo-upload" className="cursor-pointer">
                      <div className="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-8 px-3 gap-2">
                        <Upload className="w-3 h-3" />
                        {t.auth.change_photo}
                      </div>
                    </Label>
                    <input
                      id="edit-photo-upload"
                      type="file"
                      accept="image/*"
                      className="hidden"
                      onChange={(e) =>
                        e.target.files?.[0] && setPhotoBlob(e.target.files[0])
                      }
                    />
                    {photoBlob && (
                      <Button
                        variant="link"
                        size="sm"
                        type="button"
                        onClick={() => setPhotoBlob(null)}
                        className="h-8 px-2 text-destructive"
                      >
                        {t.common.cancel}
                      </Button>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="first_name"
                    render={({ field }: { field: any }) => (
                      <FormItem>
                        <FormLabel>{t.auth.first_name}</FormLabel>
                        <FormControl>
                          <Input {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="last_name"
                    render={({ field }: { field: any }) => (
                      <FormItem>
                        <FormLabel>{t.auth.last_name}</FormLabel>
                        <FormControl>
                          <Input {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="middle_name"
                    render={({ field }: { field: any }) => (
                      <FormItem>
                        <FormLabel>{t.auth.middle_name} <span className="text-muted-foreground font-normal">({t.common.optional})</span></FormLabel>
                        <FormControl>
                          <Input {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="phone"
                    render={({ field }: { field: any }) => (
                      <FormItem>
                        <FormLabel>{t.common.phone}</FormLabel>
                        <FormControl>
                          <Input {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="role"
                    render={({ field }: { field: any }) => (
                      <FormItem>
                        <FormLabel>{t.common.role}</FormLabel>
                        <Select onValueChange={field.onChange} value={field.value}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder={t.users_page.select_role} />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="user">{t.common.user}</SelectItem>
                            <SelectItem value="admin">{t.common.admin}</SelectItem>
                            <SelectItem value="superuser">{t.common.superuser || "Superuser"}</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="password"
                    render={({ field }: { field: any }) => (
                      <FormItem>
                        <FormLabel>
                          {t.profile.new_password}{" "}
                          <span className="text-muted-foreground font-normal">({t.profile.leave_blank})</span>
                        </FormLabel>
                        <FormControl>
                          <Input type="password" placeholder="••••••••" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name="telegram"
                  render={({ field }: { field: any }) => (
                    <FormItem>
                      <FormLabel>
                        {t.auth.telegram}{" "}
                        <span className="text-muted-foreground font-normal">({t.common.optional})</span>
                      </FormLabel>
                      <FormControl>
                        <Input placeholder={t.auth.telegram_placeholder} {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <DialogFooter className="pt-4">
                  <Button
                    variant="outline"
                    type="button"
                    onClick={() => onOpenChange(false)}
                    disabled={updateMutation.isPending}
                  >
                    {t.common.cancel}
                  </Button>
                  <Button type="submit" disabled={updateMutation.isPending}>
                    {updateMutation.isPending ? t.common.loading : t.common.save}
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </TabsContent>

          <TabsContent value="audit" className="mt-4">
            <ScrollArea className="h-[400px] pr-4">
              <div className="space-y-3">
                {isLoadingAudit ? (
                  <div className="flex items-center justify-center h-20 text-muted-foreground">
                    {t.common.loading}...
                  </div>
                ) : auditLogs.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-40 text-muted-foreground text-center p-4">
                    <History className="w-8 h-8 mb-2 opacity-20" />
                    <p>{t.audit_page.no_logs}</p>
                  </div>
                ) : (
                  auditLogs.map((log: AuditLog, index: number) => (
                    <div
                      key={log.id}
                      className={cn(
                        "flex items-start gap-3 p-3 rounded-lg border transition-colors",
                        index === 0 ? "bg-muted/50 border-primary/20" : "bg-card border-border hover:bg-muted/30"
                      )}
                    >
                      <div className={cn("p-2 rounded-md border", getActionColor(log.action))}>
                        {getActionIcon(log.action)}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between gap-2">
                          <p className="font-medium text-sm">{log.description}</p>
                          <span className="text-xs text-muted-foreground whitespace-nowrap">
                            {format(new Date(log.created_at), "dd MMM HH:mm", { locale: ru })}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 mt-1">
                          <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                            {log.entity}
                          </span>
                          <span className={cn(
                            "text-xs px-1.5 py-0.5 rounded",
                            log.status === 200 ? "bg-emerald-500/10 text-emerald-600" : "bg-destructive/10 text-destructive"
                          )}>
                            {log.status}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
