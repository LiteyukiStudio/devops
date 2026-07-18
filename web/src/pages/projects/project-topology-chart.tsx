import type { GraphSeriesOption } from 'echarts/charts'
import type { TooltipComponentOption } from 'echarts/components'
import type { ComposeOption, ECElementEvent, EChartsType } from 'echarts/core'
import type { ProjectTopologyEdge, ProjectTopologyNode } from '@/api'
import { GraphChart } from 'echarts/charts'
import { TooltipComponent } from 'echarts/components'
import { init, use as registerEChartsModules } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

registerEChartsModules([GraphChart, TooltipComponent, CanvasRenderer])

type ProjectTopologyChartOption = ComposeOption<GraphSeriesOption | TooltipComponentOption>

interface ProjectTopologyChartProps {
  edges: ProjectTopologyEdge[]
  fitVersion: number
  nodes: ProjectTopologyNode[]
  onSelectEdge: (edgeId: string) => void
}

export function ProjectTopologyChart({ edges, fitVersion, nodes, onSelectEdge }: ProjectTopologyChartProps) {
  const { t } = useTranslation()
  const elementRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<EChartsType | null>(null)
  const [themeVersion, setThemeVersion] = useState(0)
  const option = useMemo(() => buildChartOption(nodes, edges, t, themeVersion), [edges, nodes, t, themeVersion])

  useEffect(() => {
    const element = elementRef.current
    if (!element)
      return
    const chart = init(element)
    const resizeObserver = new ResizeObserver(() => {
      if (!chart.isDisposed())
        chart.resize()
    })
    resizeObserver.observe(element)
    chartRef.current = chart
    return () => {
      resizeObserver.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    const chart = chartRef.current
    if (!chart || chart.isDisposed())
      return
    chart.setOption(option, true)
  }, [fitVersion, option])

  useEffect(() => {
    const chart = chartRef.current
    if (!chart)
      return
    const handleClick = (event: ECElementEvent) => {
      if (event.dataType !== 'edge')
        return
      const data = event.data as { id?: string } | undefined
      if (data?.id)
        onSelectEdge(data.id)
    }
    chart.on('click', handleClick)
    return () => {
      if (!chart.isDisposed())
        chart.off('click', handleClick)
    }
  }, [onSelectEdge])

  useEffect(() => {
    const observer = new MutationObserver(() => setThemeVersion(version => version + 1))
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class', 'style'] })
    return () => observer.disconnect()
  }, [])

  return <div ref={elementRef} className="h-[64vh] min-h-104 max-h-160 w-full bg-muted/25" />
}

function buildChartOption(
  nodes: ProjectTopologyNode[],
  edges: ProjectTopologyEdge[],
  t: (key: string, options?: Record<string, unknown>) => string,
  _themeVersion: number,
): ProjectTopologyChartOption {
  const styles = getComputedStyle(document.documentElement)
  const foreground = chartThemeColor(styles, '--foreground', '#18181b')
  const muted = chartThemeColor(styles, '--foreground', 'rgba(24, 24, 27, 0.58)', 0.58)
  const border = chartThemeColor(styles, '--border', '#d4d4d8')
  const background = chartThemeColor(styles, '--surface', '#ffffff')
  const primary = chartThemeColor(styles, '--primary', '#2563eb')
  const mutedSurface = chartThemeColor(styles, '--muted', '#f4f4f5')
  const nodesById = new Map(nodes.map(node => [node.id, node]))

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      confine: true,
      backgroundColor: background,
      borderColor: border,
      textStyle: { color: foreground },
      formatter: (params) => {
        if (Array.isArray(params))
          return ''
        const data = params.data as { id?: string } | undefined
        if (params.dataType === 'node') {
          const node = data?.id ? nodesById.get(data.id) : undefined
          if (!node)
            return ''
          return [
            `<strong>${escapeHTML(node.name)}</strong>`,
            `${escapeHTML(t('projectTopology.deploymentTarget'))}: ${node.deploymentTargets.length}`,
            `${escapeHTML(t('projectTopology.relationStatus'))}: ${escapeHTML(node.status || t('common.unknown'))}`,
          ].join('<br/>')
        }
        if (params.dataType === 'edge') {
          const edge = edges.find(item => item.id === data?.id)
          if (!edge)
            return ''
          const source = nodesById.get(edge.source)?.name ?? edge.source
          const target = nodesById.get(edge.target)?.name ?? edge.target
          return [
            `<strong>${escapeHTML(t('projectTopology.direction', { source, target }))}</strong>`,
            `${escapeHTML(t('projectTopology.origin'))}: ${escapeHTML(t(`projectTopology.origins.${edge.origin}`))}`,
            `${escapeHTML(t('projectTopology.relationStatus'))}: ${escapeHTML(t(`projectTopology.statuses.${edge.status}`, { defaultValue: edge.status }))}`,
          ].join('<br/>')
        }
        return ''
      },
    },
    series: [{
      type: 'graph',
      layout: 'circular',
      circular: { rotateLabel: false },
      preserveAspect: 'contain',
      preserveAspectAlign: 'center',
      preserveAspectVerticalAlign: 'middle',
      roam: true,
      draggable: false,
      animationDurationUpdate: 280,
      data: nodes.map(node => ({
        id: node.id,
        name: truncate(node.name, 22),
        symbol: 'roundRect',
        symbolSize: [176, 66],
        itemStyle: {
          color: node.status === 'healthy' || node.status === 'ready' ? primary : mutedSurface,
          borderColor: colorForStatus(node.status, border),
          borderWidth: 2,
          shadowBlur: 8,
          shadowColor: 'rgba(0,0,0,0.08)',
        },
        label: {
          show: true,
          color: node.status === 'healthy' || node.status === 'ready' ? '#ffffff' : foreground,
          formatter: `{name|${escapeEChartsRichText(truncate(node.name, 20))}}\n{meta|${escapeEChartsRichText(t('projectTopology.relationCount', { count: degreeFor(node.id, edges) }))}}}`,
          rich: {
            name: { fontSize: 13, fontWeight: 600, lineHeight: 20 },
            meta: { fontSize: 10, opacity: 0.78, lineHeight: 16 },
          },
        },
      })),
      links: edges.map(edge => ({
        id: edge.id,
        source: edge.source,
        target: edge.target,
        lineStyle: {
          color: colorForStatus(edge.status, muted),
          curveness: 0.08,
          opacity: 0.72,
          type: edge.origin === 'manual' ? 'dashed' : 'solid',
          width: edge.origin === 'manual' ? 1.5 : 2,
        },
      })),
      edgeSymbol: ['none', 'arrow'],
      edgeSymbolSize: 9,
      emphasis: { focus: 'adjacency', lineStyle: { opacity: 1, width: 3 } },
      scaleLimit: { min: 0.4, max: 2.8 },
      left: 96,
      right: 96,
      top: 64,
      bottom: 64,
    }],
  }
}

function degreeFor(nodeId: string, edges: ProjectTopologyEdge[]) {
  return edges.filter(edge => edge.source === nodeId || edge.target === nodeId).length
}

function colorForStatus(status: string, fallback: string) {
  const normalized = status.toLowerCase()
  if (normalized.includes('fail') || normalized.includes('error') || normalized === 'invalid' || normalized === 'unavailable')
    return '#fb7185'
  if (normalized === 'ready' || normalized === 'healthy')
    return '#34d399'
  if (normalized.includes('pending'))
    return '#fbbf24'
  return fallback
}

function chartThemeColor(styles: CSSStyleDeclaration, variable: string, fallback: string, alpha?: number) {
  const value = styles.getPropertyValue(variable).trim()
  if (!value)
    return fallback
  const hslParts = value.split(/\s+/)
  if (hslParts.length === 3 && hslParts[1]?.endsWith('%') && hslParts[2]?.endsWith('%')) {
    const [hue, saturation, lightness] = hslParts
    return alpha === undefined
      ? `hsl(${hue}, ${saturation}, ${lightness})`
      : `hsla(${hue}, ${saturation}, ${lightness}, ${alpha})`
  }
  return value
}

function truncate(value: string, maxLength: number) {
  return value.length <= maxLength ? value : `${value.slice(0, maxLength - 1)}…`
}

function escapeHTML(value: string) {
  return value.replace(/[&<>'"]/g, character => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '\'': '&#39;',
    '"': '&quot;',
  })[character] ?? character)
}

function escapeEChartsRichText(value: string) {
  return value.replace(/[{}\\]/g, character => `\\${character}`)
}
