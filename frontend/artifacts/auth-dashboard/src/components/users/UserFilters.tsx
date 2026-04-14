import { Search, RefreshCw } from "lucide-react";
import { useTranslation } from "@/hooks/useTranslation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface UserFiltersProps {
  searchInput: string;
  setSearchInput: (val: string) => void;
  search: string;
  onSearchSubmit: (e: React.FormEvent) => void;
  activeFilter: string;
  onFilterChange: (val: string) => void;
  onRefresh: () => void;
  isRefetching: boolean;
}

export function UserFilters({
  searchInput,
  setSearchInput,
  search,
  onSearchSubmit,
  activeFilter,
  onFilterChange,
  onRefresh,
  isRefetching,
}: UserFiltersProps) {
  const { t } = useTranslation();

  return (
    <div className="p-4 border-b border-border/50 flex flex-col sm:flex-row gap-4 items-center justify-between bg-muted/20">
      <form
        onSubmit={onSearchSubmit}
        className="relative w-full sm:max-w-md flex items-center"
      >
        <Search className="absolute left-3 w-4 h-4 text-muted-foreground" />
        <Input
          placeholder={t.users_page.search_placeholder}
          className="pl-9 bg-background w-full"
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
        />
        {searchInput !== search && (
          <Button type="submit" variant="ghost" size="sm" className="absolute right-1 h-7">
            {t.common.search}
          </Button>
        )}
      </form>

      <div className="flex items-center gap-3 w-full sm:w-auto">
        <Select
          value={activeFilter}
          onValueChange={onFilterChange}
        >
          <SelectTrigger className="w-[160px] bg-background">
            <SelectValue placeholder={t.common.status} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t.common.all}</SelectItem>
            <SelectItem value="active">{t.common.active}</SelectItem>
            <SelectItem value="inactive">{t.common.inactive}</SelectItem>
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="icon"
          onClick={onRefresh}
          disabled={isRefetching}
          title="Обновить"
        >
          <RefreshCw className={`w-4 h-4 ${isRefetching ? "animate-spin" : ""}`} />
        </Button>
      </div>
    </div>
  );
}
