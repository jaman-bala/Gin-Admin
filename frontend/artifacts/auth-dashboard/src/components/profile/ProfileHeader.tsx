import { useRef, useState } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { ShieldCheck, Camera, Crown, Loader2 } from "lucide-react";
import { useTranslation } from "@/hooks/useTranslation";

interface ProfileHeaderProps {
  user: any;
  photoPreview: string | null;
  onPhotoChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  isPending?: boolean;
}

export function ProfileHeader({ user, photoPreview, onPhotoChange, isPending }: ProfileHeaderProps) {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [imageError, setImageError] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);

  // Сбрасываем состояние загрузки при смене src
  const handleImageLoad = () => {
    setImageError(false);
    setImageLoaded(true);
  };

  const handleImageError = () => {
    setImageError(true);
    setImageLoaded(false);
  };

  const currentSrc = photoPreview || user?.photo;

  return (
    <Card className="border-border shadow-sm overflow-hidden">
      <div className="h-32 bg-primary/10 relative w-full">
        <div className="absolute inset-0 bg-gradient-to-t from-background/80 to-transparent" />
      </div>
      <CardContent className="px-6 pb-6 pt-0 relative">
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 -mt-12 mb-6">
          <div className="flex items-end gap-5">
            <div className="relative group z-10">
              <Avatar className="w-28 h-28 border-4 border-card shadow-md bg-muted">
                {currentSrc && !imageError ? (
                  <AvatarImage
                    src={currentSrc}
                    className="object-cover"
                    onLoad={handleImageLoad}
                    onError={handleImageError}
                  />
                ) : null}
                <AvatarFallback className="text-3xl font-bold text-muted-foreground">
                  {user?.first_name?.[0]}{user?.last_name?.[0]}
                </AvatarFallback>
              </Avatar>
              <button 
                onClick={() => !isPending && fileInputRef.current?.click()}
                disabled={isPending}
                className="absolute bottom-1 right-1 w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center shadow-sm hover:scale-110 transition-transform cursor-pointer border-2 border-card disabled:opacity-50 disabled:cursor-not-allowed"
                title={t.common.edit}
              >
                {isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Camera className="w-4 h-4" />}
              </button>
              <input 
                type="file" 
                ref={fileInputRef} 
                accept="image/jpeg,image/png,image/webp" 
                className="hidden" 
                onChange={onPhotoChange}
                disabled={isPending}
              />
            </div>
            <div className="pb-1">
              <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
                {user?.first_name} {user?.last_name}
                {user?.role === "admin" && <ShieldCheck className="w-5 h-5 text-primary" />}
                {user?.role === "superuser" && <Crown className="w-5 h-5 text-amber-500" />}
              </h2>
              <p className="text-muted-foreground font-mono text-sm mt-1">{user?.id}</p>
            </div>
          </div>
          <div className="flex gap-2">
            <Badge 
              variant="outline" 
              className={`px-3 py-1 text-sm bg-muted/50 ${user?.role === "superuser" ? "text-amber-600 border-amber-500/30 bg-amber-500/10" : ""}`}
            >
              {user?.role === "superuser" ? t.common.superuser : user?.role === "admin" ? t.common.admin : t.common.user}
            </Badge>
            <Badge variant="outline" className="px-3 py-1 text-sm bg-emerald-500/10 text-emerald-600 border-emerald-500/20">
              {t.common.active}
            </Badge>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
