import { Link } from "wouter";
import { Card, CardContent } from "@/components/ui/card";
import { AlertCircle, ArrowLeft } from "lucide-react";
import { useTranslation } from "@/hooks/useTranslation";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  const { t } = useTranslation();

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-muted/30 p-4">
      <Card className="w-full max-w-md shadow-lg border-border/50">
        <CardContent className="pt-8 pb-8 flex flex-col items-center text-center">
          <div className="w-16 h-16 rounded-full bg-destructive/10 flex items-center justify-center text-destructive mb-6">
            <AlertCircle className="h-10 w-10" />
          </div>
          
          <h1 className="text-2xl font-bold tracking-tight mb-2">
            {t.not_found.title}
          </h1>
          
          <p className="text-muted-foreground mb-8">
            {t.not_found.description}
          </p>

          <Link href="/dashboard">
            <Button className="gap-2">
              <ArrowLeft className="w-4 h-4" />
              {t.not_found.back_home}
            </Button>
          </Link>
        </CardContent>
      </Card>
    </div>
  );
}
