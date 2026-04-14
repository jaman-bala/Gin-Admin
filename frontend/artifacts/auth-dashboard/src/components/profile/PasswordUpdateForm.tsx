import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useTranslation } from "@/hooks/useTranslation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { ShieldAlert, Eye, EyeOff } from "lucide-react";

interface PasswordUpdateFormProps {
  isPending: boolean;
  onSubmit: (values: any) => void;
}

// Схема валидации вынесена наружу
const passwordSchema = z.object({
  password: z.string()
    .min(8, "password_min_length")
    .regex(/[A-Z]/, "password_uppercase")
    .regex(/\d/, "password_digit"),
  confirm_password: z.string()
    .min(8, "password_min_length"),
}).refine((data) => data.password === data.confirm_password, {
  message: "passwords_mismatch",
  path: ["confirm_password"],
});

export function PasswordUpdateForm({ isPending, onSubmit }: PasswordUpdateFormProps) {
  const { t } = useTranslation();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  const form = useForm({
    resolver: zodResolver(passwordSchema),
    defaultValues: { password: "", confirm_password: "" }
  });

  const handleSubmit = async (values: any) => {
    await onSubmit(values);
    form.reset();
  };

  return (
    <Card className="border-border shadow-sm border-destructive/20">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-destructive">
          <ShieldAlert className="w-5 h-5" />
          {t.profile.change_password}
        </CardTitle>
        <CardDescription>{t.profile.password_description}</CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(handleSubmit)}>
          <CardContent className="space-y-5">
            <FormField control={form.control} name="password" render={({ field }) => (
              <FormItem>
                <FormLabel>{t.profile.new_password}</FormLabel>
                <FormControl>
                  <div className="relative flex items-center">
                    <Input type={showPassword ? "text" : "password"} {...field} disabled={isPending} />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 p-1 hover:bg-muted rounded-md transition-colors"
                      tabIndex={-1}
                    >
                      {showPassword ? <EyeOff className="w-4 h-4 text-muted-foreground" /> : <Eye className="w-4 h-4 text-muted-foreground" />}
                    </button>
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="confirm_password" render={({ field }) => (
              <FormItem>
                <FormLabel>{t.profile.confirm_new_password}</FormLabel>
                <FormControl>
                  <div className="relative flex items-center">
                    <Input type={showConfirmPassword ? "text" : "password"} {...field} disabled={isPending} />
                    <button
                      type="button"
                      onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                      className="absolute right-3 p-1 hover:bg-muted rounded-md transition-colors"
                      tabIndex={-1}
                    >
                      {showConfirmPassword ? <EyeOff className="w-4 h-4 text-muted-foreground" /> : <Eye className="w-4 h-4 text-muted-foreground" />}
                    </button>
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )} />
          </CardContent>
          <CardFooter className="border-t border-destructive/10 bg-destructive/5 pt-5 justify-end">
            <Button variant="destructive" type="submit" disabled={isPending || !form.formState.isDirty}>
              {t.profile.change_password}
            </Button>
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
}
