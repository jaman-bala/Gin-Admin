import React from "react";
import { cn } from "@/lib/utils";

interface LogoProps {
  className?: string;
  iconClassName?: string;
  showText?: boolean;
  size?: "sm" | "md" | "lg";
}

const sizeMap = {
  sm: { icon: 24, fontSize: "text-base", gap: "gap-2" },
  md: { icon: 36, fontSize: "text-xl", gap: "gap-3" },
  lg: { icon: 48, fontSize: "text-2xl", gap: "gap-4" },
};

export function Logo({ className, iconClassName, showText = true, size = "md" }: LogoProps) {
  const { icon, fontSize, gap } = sizeMap[size];

  return (
    <div className={cn("group flex items-center transition-all duration-300", gap, className)}>
      {/* Premium SVG Icon */}
      <div className="relative flex items-center justify-center">
        <svg
          width={icon}
          height={icon}
          viewBox="0 0 32 32"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          className={cn(
            "drop-shadow-[0_0_8px_rgba(79,70,229,0.3)] transition-all duration-500 group-hover:scale-110 group-hover:drop-shadow-[0_0_12px_rgba(6,182,212,0.5)]",
            iconClassName
          )}
        >
          <defs>
            <linearGradient id="logo-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#4F46E5" />
              <stop offset="100%" stopColor="#06B6D4" />
            </linearGradient>
            <filter id="glow">
              <feGaussianBlur stdDeviation="1.5" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
          </defs>
          
          {/* Main "G" Shape */}
          <path
            d="M26 12C26 12 24.5 7 19.5 5C14.5 3 8 5.5 6 12.5C4 19.5 8 26.5 16 27.5C24 28.5 27.5 22 27.5 16H16"
            stroke="url(#logo-gradient)"
            strokeWidth="3.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          
          {/* Inner accent point */}
          <circle
            cx="16"
            cy="16"
            r="2.5"
            fill="url(#logo-gradient)"
            className="animate-pulse"
          />
        </svg>

        {/* Backdrop Glow */}
        <div className="absolute -z-10 h-2/3 w-2/3 rounded-full bg-primary/20 blur-xl transition-all duration-500 group-hover:bg-primary/40 group-hover:blur-2xl" />
      </div>

      {showText && (
        <div className={cn("flex flex-col leading-none transition-colors duration-300", fontSize)}>
          <span className="font-black tracking-tighter text-foreground group-hover:text-primary transition-colors">
            GIN
          </span>
          <span className="text-[0.6em] font-light uppercase tracking-[0.3em] text-muted-foreground/80">
            Admin
          </span>
        </div>
      )}
    </div>
  );
}
