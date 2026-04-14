import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useTranslation } from "@/hooks/useTranslation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Phone } from "lucide-react";

interface PhoneUpdateFormProps {
  user: any;
  isPending: boolean;
  onSubmit: (values: any) => void;
}

// Схема валидации вынесена наружу
const phoneSchema = z.object({
  phone: z.string().regex(/^\+[1-9]\d{1,14}$/, "phone_invalid"),
});

export function PhoneUpdateForm({ user, isPending, onSubmit }: PhoneUpdateFormProps) {
  const { t } = useTranslation();

  const form = useForm({
    resolver: zodResolver(phoneSchema),
    values: {
      phone: user?.phone || "",
    }
  });

  return (
    <Card className="border-border shadow-sm">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Phone className="w-5 h-5 text-primary" />
          {t.profile.login_credentials}
        </CardTitle>
        <CardDescription>{t.profile.phone_description}</CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="space-y-5">
            <FormField control={form.control} name="phone" render={({ field }) => (
              <FormItem>
                <FormLabel>{t.common.phone}</FormLabel>
                <FormControl><Input {...field} disabled={isPending} /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
          </CardContent>
          <CardFooter className="border-t border-border/50 bg-muted/10 pt-5 justify-end">
            <Button type="submit" variant="outline" disabled={isPending || !form.formState.isDirty}>
              {isPending && form.formState.isDirty ? t.common.loading : t.common.update_phone}
            </Button>
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
}
