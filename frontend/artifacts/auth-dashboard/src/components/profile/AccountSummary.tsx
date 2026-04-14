import { format } from "date-fns";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Check, ShieldCheck, Moon, Sun, Monitor } from "lucide-react";
import { useTranslation } from "@/hooks/useTranslation";
import { useTheme } from "@/hooks/use-theme";
import { Button } from "@/components/ui/button";

interface AccountSummaryProps {
  user: any;
}

export function AccountSummary({ user }: AccountSummaryProps) {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();

  return (
    <Card className="border-border shadow-sm">
      <CardHeader>
        <CardTitle className="text-lg">{t.profile.account_summary}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <Label className="text-xs text-muted-foreground uppercase tracking-wider mb-1 block">
            {t.profile.account_id}
          </Label>
          <p className="text-sm font-mono break-all bg-muted/50 p-2 rounded border border-border/50">
            {user?.id}
          </p>
        </div>
        <Separator />
        <div>
          <Label className="text-xs text-muted-foreground uppercase tracking-wider mb-1 block">
            {t.profile.role_privileges}
          </Label>
          <div className="flex items-center gap-2 mt-2">
            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary">
              <Check className="w-4 h-4" />
            </div>
            <span className="text-sm font-medium">{t.profile.basic_access}</span>
          </div>
          {user?.role === "admin" && (
            <div className="flex items-center gap-2 mt-2">
              <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary">
                <ShieldCheck className="w-4 h-4" />
              </div>
              <span className="text-sm font-medium">{t.profile.system_admin}</span>
            </div>
          )}
          {user?.role === "superuser" && (
            <div className="flex items-center gap-2 mt-2">
              <div className="w-8 h-8 rounded-full bg-amber-500/10 flex items-center justify-center text-amber-500">
                <ShieldCheck className="w-4 h-4" />
              </div>
              <span className="text-sm font-medium">{t.profile.superuser_access}</span>
            </div>
          )}
        </div>
        <Separator />
        <div>
          <Label className="text-xs text-muted-foreground uppercase tracking-wider mb-1 block">
            {t.users_page.joined}
          </Label>
          <p className="text-sm font-medium">
            {user?.created_at ? format(new Date(user.created_at), "d MMMM yyyy") : "—"}
          </p>
        </div>
        <div>
          <Label className="text-xs text-muted-foreground uppercase tracking-wider mb-1 block">
            {t.profile.last_update}
          </Label>
          <p className="text-sm font-medium">
            {user?.updated_at ? format(new Date(user.updated_at), "d MMMM yyyy 'в' HH:mm") : "—"}
          </p>
        </div>
        <Separator />
        <div>
          <Label className="text-xs text-muted-foreground uppercase tracking-wider mb-3 block">
            {t.profile.appearance}
          </Label>
          <div className="flex gap-2">
            <Button
              variant={theme === "light" ? "default" : "outline"}
              size="sm"
              onClick={() => setTheme("light")}
              className="flex-1 gap-1.5 px-2 py-1 h-8 text-xs"
            >
              <Sun className="w-3.5 h-3.5" />
              {t.profile.theme_light}
            </Button>
            <Button
              variant={theme === "dark" ? "default" : "outline"}
              size="sm"
              onClick={() => setTheme("dark")}
              className="flex-1 gap-1.5 px-2 py-1 h-8 text-xs"
            >
              <Moon className="w-3.5 h-3.5" />
              {t.profile.theme_dark}
            </Button>
            <Button
              variant={theme === "system" ? "default" : "outline"}
              size="sm"
              onClick={() => setTheme("system")}
              className="flex-1 gap-1.5 px-2 py-1 h-8 text-xs"
            >
              <Monitor className="w-3.5 h-3.5" />
              {t.profile.theme_system}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
