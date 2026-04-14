import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useGetMe, getGetMeQueryKey, useUpdateMe } from "@workspace/api-client-react";
import { useAuthStore } from "@/store/authStore";
import { useTranslation } from "@/hooks/useTranslation";
import { useToast } from "@/hooks/use-toast";
import { Skeleton } from "@/components/ui/skeleton";
import { ShieldAlert } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

// Modular Components
import { ProfileHeader } from "@/components/profile/ProfileHeader";
import { PersonalInfoForm } from "@/components/profile/PersonalInfoForm";
import { PhoneUpdateForm } from "@/components/profile/PhoneUpdateForm";
import { PasswordUpdateForm } from "@/components/profile/PasswordUpdateForm";
import { AccountSummary } from "@/components/profile/AccountSummary";

export default function Profile() {
  const { t } = useTranslation();
  const { user: storeUser, setUser } = useAuthStore();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  
  const { data: me, isLoading, refetch: refetchMe } = useGetMe({ 
    query: { queryKey: getGetMeQueryKey() } 
  });

  const updateMutation = useUpdateMe({
    mutation: {
      onSuccess: async (data) => {
        toast({ title: t.profile.update_success, description: t.common.success });
        // Refetch to get updated photo URL with fresh signature
        const { data: freshData } = await refetchMe();
        if (freshData) {
          setUser(freshData);
          queryClient.setQueryData(getGetMeQueryKey(), freshData);
        }
        // Не сбрасываем preview сразу — пусть пользователь видит результат
        // Сбросим только blob, preview сбросится когда фото загрузится или при следующей смене
        setPhotoBlob(null);
      },
      onError: (err: any) => {
        toast({
          title: t.common.error,
          description: err.response?.data?.message || err.response?.data?.error || t.common.error,
          variant: "destructive"
        });
      }
    }
  });

  const user = me || storeUser;

  const [photoBlob, setPhotoBlob] = useState<Blob | null>(null);
  const [photoPreview, setPhotoPreview] = useState<string | null>(null);
  const [isConfirmDialogOpen, setIsConfirmDialogOpen] = useState(false);
  const [pendingPhoneValue, setPendingPhoneValue] = useState<string | null>(null);

  const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5MB
  const ALLOWED_TYPES = ["image/jpeg", "image/png", "image/webp"];

  const handlePhotoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Валидация размера
    if (file.size > MAX_FILE_SIZE) {
      toast({
        title: t.common.error,
        description: t.profile.file_too_large,
        variant: "destructive"
      });
      e.target.value = "";
      return;
    }

    // Валидация типа
    if (!ALLOWED_TYPES.includes(file.type)) {
      toast({
        title: t.common.error,
        description: t.profile.file_invalid_type,
        variant: "destructive"
      });
      e.target.value = "";
      return;
    }

    setPhotoBlob(file);
    const reader = new FileReader();
    reader.onloadend = () => {
      setPhotoPreview(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  const onProfileSubmit = (values: any) => {
    const dataToSubmit: any = { ...values };
    if (photoBlob) dataToSubmit.photo = photoBlob;
    updateMutation.mutate({ data: dataToSubmit });
  };

  const onPhoneSubmit = (values: any) => {
    if (values.phone === user?.phone) return;
    setPendingPhoneValue(values.phone);
    setIsConfirmDialogOpen(true);
  };

  const confirmPhoneUpdate = () => {
    if (pendingPhoneValue) {
      updateMutation.mutate({ data: { phone: pendingPhoneValue } }, {
        onSuccess: () => {
          setIsConfirmDialogOpen(false);
          setPendingPhoneValue(null);
        }
      });
    }
  };

  const onPasswordSubmit = async (values: any) => {
    await updateMutation.mutateAsync({ data: { password: values.password } });
    // Toast вызывается в onSuccess updateMutation
  };

  if (isLoading && !user) {
    return (
      <div className="space-y-6 max-w-4xl mx-auto">
        <Skeleton className="h-10 w-48 mb-2" />
        <Skeleton className="h-64 rounded-xl" />
        <Skeleton className="h-96 rounded-xl" />
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-4xl mx-auto animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t.profile.title}</h1>
        <p className="text-muted-foreground mt-1">{t.profile.description}</p>
      </div>

      <ProfileHeader 
        user={user} 
        photoPreview={photoPreview} 
        onPhotoChange={handlePhotoChange}
        isPending={updateMutation.isPending}
      />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <PersonalInfoForm 
            user={user} 
            isPending={updateMutation.isPending} 
            hasPhoto={!!photoBlob}
            onSubmit={onProfileSubmit} 
          />

          <PhoneUpdateForm 
            user={user} 
            isPending={updateMutation.isPending} 
            onSubmit={onPhoneSubmit} 
          />

          <PasswordUpdateForm 
            isPending={updateMutation.isPending} 
            onSubmit={onPasswordSubmit} 
          />
        </div>

        <div className="space-y-6">
          <AccountSummary user={user} />
        </div>
      </div>

      <AlertDialog open={isConfirmDialogOpen} onOpenChange={setIsConfirmDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <ShieldAlert className="w-5 h-5 text-destructive" />
              {t.profile.phone_update_warning}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t.profile.phone_update_description}
              <br /><br />
              {t.profile.new_phone_label}: <span className="font-bold text-foreground">{pendingPhoneValue}</span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t.common.cancel}</AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmPhoneUpdate}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t.common.save}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
