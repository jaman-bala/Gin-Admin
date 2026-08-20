import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useTranslation } from "@/hooks/useTranslation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";

interface PersonalInfoFormProps {
  user: any;
  isPending: boolean;
  hasPhoto: boolean;
  onSubmit: (values: any) => void;
}

// Схема валидации вынесена наружу
const profileSchema = z.object({
  first_name: z.string().min(1, "first_name_required"),
  last_name: z.string().min(1, "last_name_required"),
  middle_name: z.string().optional(),
  telegram: z.string().optional(),
});

export function PersonalInfoForm({ user, isPending, hasPhoto, onSubmit }: PersonalInfoFormProps) {
  const { t } = useTranslation();

  const form = useForm({
    resolver: zodResolver(profileSchema),
    values: {
      first_name: user?.first_name || "",
      last_name: user?.last_name || "",
      middle_name: user?.middle_name || "",
      telegram: user?.telegram || "",
    }
  });

  return (
    <Card className="border-border shadow-sm">
      <CardHeader>
        <CardTitle>{t.profile.personal_info}</CardTitle>
        <CardDescription>{t.profile.personal_description}</CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="space-y-5">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
              <FormField control={form.control} name="first_name" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t.auth.first_name}</FormLabel>
                  <FormControl><Input {...field} disabled={isPending} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="last_name" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t.auth.last_name}</FormLabel>
                  <FormControl><Input {...field} disabled={isPending} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
            </div>
            <FormField control={form.control} name="middle_name" render={({ field }) => (
              <FormItem>
                <FormLabel>{t.auth.middle_name} <span className="text-muted-foreground font-normal">({t.common.optional})</span></FormLabel>
                <FormControl><Input {...field} disabled={isPending} /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="telegram" render={({ field }) => (
              <FormItem>
                <FormLabel>{t.auth.telegram} <span className="text-muted-foreground font-normal">({t.common.optional})</span></FormLabel>
                <FormControl><Input placeholder={t.auth.telegram_placeholder} {...field} disabled={isPending} /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
          </CardContent>
          <CardFooter className="border-t border-border/50 bg-muted/10 pt-5 justify-end">
            <Button type="submit" disabled={isPending || (!form.formState.isDirty && !hasPhoto)}>
              {isPending && form.formState.isDirty ? t.common.loading : t.common.save}
            </Button>
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
}
