import { useState, useEffect } from "react";
import { useLocation } from "wouter";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useAuthLogin } from "@workspace/api-client-react";
import { useAuthStore } from "@/store/authStore";
import { useToast } from "@/hooks/use-toast";
import { useTranslation } from "@/hooks/useTranslation";
import { Button } from "@/components/ui/button";
import { Eye, EyeOff, Loader2 } from "lucide-react";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Logo } from "@/components/layout/Logo";

// Схема валидации — вынесена за пределы компонента
const loginSchema = z.object({
  phone: z.string().regex(/^\+[1-9]\d{1,14}$/, {
    message: "Введите телефон в формате +996500500500"
  }),
  password: z.string()
    .min(8, { message: "Пароль должен быть не менее 8 символов" })
    .regex(/[A-Z]/, { message: "Пароль должен содержать заглавную букву" })
    .regex(/[a-z]/, { message: "Пароль должен содержать строчную букву" })
    .regex(/\d/, { message: "Пароль должен содержать цифру" }),
});

type LoginFormValues = z.infer<typeof loginSchema>;

export default function Login() {
  const { t } = useTranslation();
  const [, setLocation] = useLocation();
  const { setAuth } = useAuthStore();
  const { toast } = useToast();
  const [error, setError] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  const loginMutation = useAuthLogin();

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      phone: "",
      password: "",
    },
  });

  // Сбрасываем ошибку при изменении любого поля
  const watchAllFields = form.watch();
  useEffect(() => {
    if (error) setError("");
  }, [watchAllFields]);

  const onSubmit = (values: LoginFormValues) => {
    setError("");
    loginMutation.mutate({ data: values }, {
      onSuccess: async (data) => {
        try {
          // Прямой запрос getMe с токеном в заголовке — без race condition
          const userData = await fetch('/api/auth/me', {
            headers: {
              'Authorization': `Bearer ${data.access_token}`,
              'Content-Type': 'application/json'
            }
          }).then(r => {
            if (!r.ok) throw new Error('Failed to get user');
            return r.json();
          });

          // Сохраняем всё в стор
          setAuth(userData, data.access_token, data.refresh_token);
          toast({ title: t.common.success, description: t.auth.sign_in });
          setLocation("/dashboard");
        } catch (e) {
          setError(t.common.error);
        }
      },
      onError: (err: any) => {
        const msg = err.response?.data?.message || err.response?.data?.error || err.message || t.common.error;
        setError(msg);
      }
    });
  };

  return (
    <div className="min-h-screen bg-muted/30 flex items-center justify-center p-4 relative overflow-hidden">
      {/* Декоративные элементы */}
      <div className="absolute top-0 left-0 w-full h-96 bg-primary/5 [mask-image:linear-gradient(to_bottom,white,transparent)] pointer-events-none" />
      <div className="absolute -top-24 -right-24 w-96 h-96 bg-primary/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute -bottom-24 -left-24 w-96 h-96 bg-primary/10 rounded-full blur-3xl pointer-events-none" />

      <div className="w-full max-w-md z-10">
        <div className="flex flex-col items-center mb-8 text-center">
          <Logo 
            showText={false} 
            size="lg"
            iconClassName="w-16 h-16" 
            className="mb-6 drop-shadow-2xl shadow-primary/20" 
          />
          <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.auth.login_title}</h1>
          <p className="text-muted-foreground mt-2">{t.auth.login_description}</p>
        </div>

        <Card className="border-border/50 shadow-xl shadow-black/5">
          <CardContent className="pt-6">
            {error && (
              <div className="mb-6 p-4 rounded-md bg-destructive/10 text-destructive border border-destructive/20 text-sm font-medium animate-in fade-in zoom-in-95">
                {error}
              </div>
            )}

            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
                <FormField
                  control={form.control}
                  name="phone"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t.common.phone}</FormLabel>
                      <FormControl>
                        <Input 
                          placeholder={t.auth.phone_placeholder} 
                          {...field} 
                          className="h-11" 
                          disabled={loginMutation.isPending} 
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                
                <FormField
                  control={form.control}
                  name="password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t.common.password}</FormLabel>
                      <FormControl>
                        <div className="relative flex items-center">
                          <Input 
                            type={showPassword ? "text" : "password"} 
                            placeholder={t.auth.password_placeholder} 
                            {...field} 
                            className="h-11 pr-10"
                            disabled={loginMutation.isPending} 
                          />
                          <button
                            type="button"
                            className="absolute right-3 text-muted-foreground hover:text-foreground transition-colors focus:outline-none"
                            onClick={() => setShowPassword(!showPassword)}
                            disabled={loginMutation.isPending}
                            tabIndex={-1}
                          >
                            {showPassword ? (
                              <EyeOff className="h-4 w-4" />
                            ) : (
                              <Eye className="h-4 w-4" />
                            )}
                          </button>
                        </div>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button 
                  type="submit" 
                  className="w-full h-11 text-base font-medium shadow-md shadow-primary/20 transition-all active:scale-[0.98]" 
                  disabled={loginMutation.isPending}
                >
                  {loginMutation.isPending ? (
                    <span className="flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      {t.common.loading}
                    </span>
                  ) : (
                    t.auth.sign_in
                  )}
                </Button>
              </form>
            </Form>
          </CardContent>
          <CardFooter className="flex justify-center pb-6 border-t border-border/50 pt-6 mt-2 bg-muted/20">
            <p className="text-sm text-muted-foreground text-center">
              {t.auth.contact_admin || "Обратитесь к администратору для создания учетной записи"}
            </p>
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}