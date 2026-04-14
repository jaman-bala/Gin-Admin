import { useState, useRef } from "react";
import { useLocation, Link } from "wouter";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useAuthRegister } from "@workspace/api-client-react";
import { useToast } from "@/hooks/use-toast";
import { useTranslation } from "@/hooks/useTranslation";
import { Button } from "@/components/ui/button";
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
import { Upload, X, User as UserIcon } from "lucide-react";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";

export default function Register() {
  const { t } = useTranslation();
  const [, setLocation] = useLocation();
  const { toast } = useToast();
  const [error, setError] = useState("");
  const [photoBlob, setPhotoBlob] = useState<Blob | null>(null);
  const [photoPreview, setPhotoPreview] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const registerSchema = z.object({
    first_name: z.string().min(1, t.auth.first_name),
    last_name: z.string().min(1, t.auth.last_name),
    middle_name: z.string().optional(),
    phone: z.string().min(5, t.common.phone),
    password: z.string().min(6, t.common.password),
  });

  type RegisterFormValues = z.infer<typeof registerSchema>;

  const registerMutation = useAuthRegister();

  const form = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      first_name: "",
      last_name: "",
      middle_name: "",
      phone: "",
      password: "",
    },
  });

  const handlePhotoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setPhotoBlob(file);
      const reader = new FileReader();
      reader.onloadend = () => {
        setPhotoPreview(reader.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const clearPhoto = () => {
    setPhotoBlob(null);
    setPhotoPreview(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const onSubmit = (values: RegisterFormValues) => {
    setError("");
    const dataToSubmit = {
      ...values,
      ...(photoBlob ? { photo: photoBlob } : {}),
    };

    registerMutation.mutate({ data: dataToSubmit }, {
      onSuccess: () => {
        toast({ title: t.common.success, description: "Регистрация успешна! Теперь вы можете войти." });
        setLocation("/login");
      },
      onError: (err: any) => {
        const msg = err.response?.data?.message || err.response?.data?.error || err.message || t.common.error;
        setError(msg);
      }
    });
  };

  return (
    <div className="min-h-screen bg-muted/30 flex items-center justify-center p-4 relative overflow-hidden py-12">
      {/* Decorative background elements */}
      <div className="absolute top-0 right-0 w-full h-96 bg-primary/5 [mask-image:linear-gradient(to_bottom,white,transparent)] pointer-events-none" />
      <div className="absolute top-1/4 -left-24 w-96 h-96 bg-primary/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-0 right-0 w-96 h-96 bg-primary/10 rounded-full blur-3xl pointer-events-none" />

      <div className="w-full max-w-xl z-10">
        <div className="flex flex-col items-center mb-8 text-center">
          <Logo 
            showText={false} 
            size="lg"
            iconClassName="w-16 h-16" 
            className="mb-6 drop-shadow-2xl shadow-primary/20" 
          />
          <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.auth.register_title}</h1>
          <p className="text-muted-foreground mt-2">{t.auth.register_description}</p>
        </div>

        <Card className="border-border/50 shadow-xl shadow-black/5">
          <CardContent className="pt-6">
            {error && (
              <div className="mb-6 p-4 rounded-md bg-destructive/10 text-destructive border border-destructive/20 text-sm font-medium">
                {error}
              </div>
            )}

            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                
                {/* Photo Upload Section */}
                <div className="flex flex-col items-center justify-center space-y-4 pb-4 border-b border-border/50">
                  <div className="relative group">
                    <Avatar className="w-24 h-24 border-2 border-border shadow-sm">
                      <AvatarImage src={photoPreview || undefined} />
                      <AvatarFallback className="bg-muted text-muted-foreground text-xl font-medium">
                        {form.watch("first_name")?.[0] || ""}{form.watch("last_name")?.[0] || ""}
                        {!form.watch("first_name") && !form.watch("last_name") && <UserIcon className="w-8 h-8 opacity-50" />}
                      </AvatarFallback>
                    </Avatar>
                    
                    {photoPreview ? (
                      <button
                        type="button"
                        onClick={clearPhoto}
                        className="absolute -top-2 -right-2 bg-destructive text-destructive-foreground rounded-full p-1 shadow-sm opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    ) : null}
                  </div>
                  
                  <div>
                    <input
                      type="file"
                      accept="image/*"
                      className="hidden"
                      ref={fileInputRef}
                      onChange={handlePhotoChange}
                    />
                    <Button 
                      type="button" 
                      variant="outline" 
                      size="sm" 
                      onClick={() => fileInputRef.current?.click()}
                      className="gap-2"
                    >
                      <Upload className="w-4 h-4" />
                      {photoPreview ? t.auth.change_photo : t.auth.upload_photo}
                    </Button>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                  <FormField
                    control={form.control}
                    name="first_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t.auth.first_name}</FormLabel>
                        <FormControl>
                          <Input placeholder="Иван" {...field} disabled={registerMutation.isPending} />
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
                          <Input placeholder="Иванов" {...field} disabled={registerMutation.isPending} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name="middle_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t.auth.middle_name} <span className="text-muted-foreground font-normal">({t.common.optional})</span></FormLabel>
                      <FormControl>
                        <Input placeholder="Иванович" {...field} disabled={registerMutation.isPending} />
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
                        <Input placeholder={t.auth.phone_placeholder} {...field} disabled={registerMutation.isPending} />
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
                        <Input type="password" placeholder={t.auth.password_placeholder} {...field} disabled={registerMutation.isPending} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button type="submit" className="w-full h-11 text-base font-medium shadow-md shadow-primary/20 transition-all hover:shadow-lg hover:shadow-primary/30 mt-4" disabled={registerMutation.isPending}>
                  {registerMutation.isPending ? t.common.loading : t.auth.sign_up}
                </Button>
              </form>
            </Form>
          </CardContent>
          <CardFooter className="flex justify-center pb-6 border-t border-border/50 pt-6 mt-2 bg-muted/20">
            <p className="text-sm text-muted-foreground">
              {t.auth.have_account}{" "}
              <Link href="/login" className="font-semibold text-primary hover:underline underline-offset-4">
                {t.auth.sign_in}
              </Link>
            </p>
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}
