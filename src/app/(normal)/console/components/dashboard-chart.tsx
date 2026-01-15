"use client"

import * as React from "react"
import { cn } from "@/utils"

interface ChartDataPoint {
  name: string
  value: number
  secondary?: number
}

interface DashboardChartProps {
  title: string
  data: ChartDataPoint[]
  type?: "bar" | "line"
  height?: number
  className?: string
}

export function DashboardChart({
  title,
  data,
  type = "bar",
  height = 200,
  className,
}: DashboardChartProps) {
  const maxValue = Math.max(...data.map((d) => d.value))

  return (
    <div
      className={cn(
        "rounded-xl border border-border-subtle bg-bg-surface p-6",
        "hover:shadow-premium transition-shadow duration-300",
        className
      )}
    >
      <h3 className="mb-4 text-sm font-medium text-text-muted">{title}</h3>

      {type === "bar" ? (
        <div className="flex items-end gap-2" style={{ height }}>
          {data.map((item, index) => {
            const barHeight = (item.value / maxValue) * 100
            return (
              <div
                key={index}
                className="group relative flex flex-1 flex-col items-center"
              >
                {/* Tooltip */}
                <div className="absolute -top-8 z-10 hidden rounded bg-bg-canvas px-2 py-1 text-xs font-medium shadow-lg group-hover:block">
                  {item.value}
                </div>
                
                {/* Bar */}
                <div
                  className="w-full rounded-t-md bg-linear-to-t from-primary to-primary/60 transition-all duration-500 ease-out group-hover:from-primary group-hover:to-primary/80"
                  style={{
                    height: `${barHeight}%`,
                    animationDelay: `${index * 50}ms`,
                  }}
                />
                
                {/* Label */}
                <span className="mt-2 text-xs text-text-muted">{item.name}</span>
              </div>
            )
          })}
        </div>
      ) : (
        <div className="relative" style={{ height }}>
          {/* Line chart using SVG */}
          <svg className="h-full w-full" viewBox={`0 0 ${data.length * 50} ${height}`}>
            {/* Grid lines */}
            {[0, 25, 50, 75, 100].map((percent) => (
              <line
                key={percent}
                x1="0"
                y1={height - (height * percent) / 100}
                x2={data.length * 50}
                y2={height - (height * percent) / 100}
                stroke="currentColor"
                strokeOpacity="0.1"
                strokeDasharray="4 4"
              />
            ))}
            
            {/* Area fill */}
            <path
              d={`
                M 0 ${height}
                ${data.map((item, i) => {
                  const x = i * 50 + 25
                  const y = height - (item.value / maxValue) * (height - 20)
                  return `L ${x} ${y}`
                }).join(" ")}
                L ${data.length * 50} ${height}
                Z
              `}
              className="fill-primary/10"
            />
            
            {/* Line */}
            <path
              d={data
                .map((item, i) => {
                  const x = i * 50 + 25
                  const y = height - (item.value / maxValue) * (height - 20)
                  return `${i === 0 ? "M" : "L"} ${x} ${y}`
                })
                .join(" ")}
              fill="none"
              className="stroke-primary"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
            
            {/* Data points */}
            {data.map((item, i) => {
              const x = i * 50 + 25
              const y = height - (item.value / maxValue) * (height - 20)
              return (
                <g key={i}>
                  <circle
                    cx={x}
                    cy={y}
                    r="4"
                    className="fill-bg-surface stroke-primary"
                    strokeWidth="2"
                  />
                  <text
                    x={x}
                    y={height - 5}
                    textAnchor="middle"
                    className="fill-text-muted text-[10px]"
                  >
                    {item.name}
                  </text>
                </g>
              )
            })}
          </svg>
        </div>
      )}
    </div>
  )
}

// Empty state
export function ChartEmptyState({ message = "No data available" }: { message?: string }) {
  return (
    <div className="flex h-48 items-center justify-center rounded-xl border border-dashed border-border-subtle bg-bg-subtle/30">
      <p className="text-sm text-text-muted">{message}</p>
    </div>
  )
}
