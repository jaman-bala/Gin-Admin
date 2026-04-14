import * as React from "react"
import { 
  Pagination, 
  PaginationContent, 
  PaginationEllipsis, 
  PaginationItem, 
  PaginationLink, 
  PaginationNext, 
  PaginationPrevious 
} from "@/components/ui/pagination"
import { translations } from "@/lib/translations"

interface PaginationControlProps {
  currentPage: number
  totalCount: number
  pageSize: number
  onPageChange: (page: number) => void
  className?: string
}

export function PaginationControl({
  currentPage,
  totalCount,
  pageSize,
  onPageChange,
  className,
}: PaginationControlProps) {
  const totalPages = Math.ceil(totalCount / pageSize)
  const t = translations.common

  if (totalPages <= 1) return null

  const getPages = () => {
    const pages: (number | string)[] = []
    const delta = 2

    for (let i = 1; i <= totalPages; i++) {
      if (
        i === 1 ||
        i === totalPages ||
        (i >= currentPage - delta && i <= currentPage + delta)
      ) {
        pages.push(i)
      } else if (
        (i === currentPage - delta - 1 || i === currentPage + delta + 1) &&
        !pages.includes("ellipsis")
      ) {
        pages.push("ellipsis")
      }
    }

    return pages
  }

  return (
    <Pagination className={className}>
      <PaginationContent>
        <PaginationItem>
          <PaginationPrevious
            href="#"
            onClick={(e) => {
              e.preventDefault()
              if (currentPage > 1) onPageChange(currentPage - 1)
            }}
            className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
          >
            {t.previous}
          </PaginationPrevious>
        </PaginationItem>

        {getPages().map((page, index) => (
          <PaginationItem key={index}>
            {page === "ellipsis" ? (
              <PaginationEllipsis />
            ) : (
              <PaginationLink
                href="#"
                isActive={currentPage === page}
                onClick={(e) => {
                  e.preventDefault()
                  onPageChange(page as number)
                }}
                className="cursor-pointer"
              >
                {page}
              </PaginationLink>
            )}
          </PaginationItem>
        ))}

        <PaginationItem>
          <PaginationNext
            href="#"
            onClick={(e) => {
              e.preventDefault()
              if (currentPage < totalPages) onPageChange(currentPage + 1)
            }}
            className={currentPage === totalPages ? "pointer-events-none opacity-50" : "cursor-pointer"}
          >
            {t.next}
          </PaginationNext>
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  )
}

