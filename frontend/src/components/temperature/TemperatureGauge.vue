<script setup lang="ts">
  import { computed, ref } from 'vue'

  import type { FloatTimeValue } from '../../utils'

  const props = defineProps<{
    temperatureValue?: FloatTimeValue
    minTemp: number
    maxTemp: number
    widthFactor?: number
    heightFactor?: number
    fontSizeFactor?: number
  }>()

  const containerRef = ref<HTMLElement | null>(null)

  const widthFactor = computed(() => props.widthFactor ?? 1)
  const heightFactor = computed(() => props.heightFactor ?? 1)
  const fontSizeFactor = computed(() => props.fontSizeFactor ?? 1)

  const containerStyle = computed(() => ({
    width: `${widthFactor.value * 100}%`,
    height: `${heightFactor.value * 100}%`,
  }))

  const option = computed(() => {
    const viewportWidth = Math.max(document.documentElement.clientWidth || 0, window.innerWidth || 0)
    const fontSize = viewportWidth > 600 ? 1.35 * fontSizeFactor.value : 0.7 * fontSizeFactor.value
    const value = ((props.temperatureValue?.value ?? 0) * 100) / props.maxTemp
    const color = getTempColor(props.temperatureValue)

    return {
      backgroundColor: 'transparent',
      series: [
        {
          type: 'gauge',
          startAngle: 200,
          endAngle: -20,
          min: 0,
          max: 100,
          radius: '100%',
          center: ['50%', '60%'],
          axisLine: {
            lineStyle: {
              width: 16,
              color: [[1, 'rgba(255, 255, 255, 0.0)']],
            },
          },
          pointer: {
            show: false,
          },
          progress: {
            show: true,
            width: 16,
            roundCap: true,
            itemStyle: {
              color: color,
            },
          },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { show: false },
          detail: {
            valueAnimation: false,
            formatter: () => (props.temperatureValue?.value ?? 0).toFixed(1),
            color: '#ffffff',
            fontFamily: 'Seven Segment',
            fontSize: fontSize * 16,
            offsetCenter: [0, '25%'],
          },
          data: [{ value }],
        },
      ],
    }
  })

  function getTempRoomsGradient(value: number, minValue: number, maxValue: number): string {
    const clamped = Math.max(minValue, Math.min(value, maxValue))
    const ratio = (clamped - minValue) / (maxValue - minValue)
    const r = Math.round(255 * ratio)
    const g = 0
    const b = Math.round(255 * (1 - ratio))
    return `rgb(${r}, ${g}, ${b})`
  }

  function getTempColor(temp?: FloatTimeValue): string {
    if (!temp) return 'rgb(255, 255, 255)'
    const hoursPrevious24 = new Date()
    hoursPrevious24.setHours(hoursPrevious24.getHours() - 24)
    if (temp.timestamp < hoursPrevious24) return 'rgb(255, 255, 0)'
    return getTempRoomsGradient(temp.value, props.minTemp, props.maxTemp)
  }
</script>

<template>
  <div ref="containerRef" :style="containerStyle">
    <VChart :option="option" autoresize style="width: 100%; height: 100%" />
  </div>
</template>
