import type { InjectionKey } from 'vue'

export type OpenChartFn = (tagId: string, title: string, lowerLimit?: number, upperLimit?: number) => void

/** Provided by MainView — opens a plain line chart */
export const openChartKey: InjectionKey<OpenChartFn> = Symbol('openChart')

/** Provided by MainView — opens a time-series scatter chart */
export const openChartTimeSeriesKey: InjectionKey<OpenChartFn> = Symbol('openChartTimeSeries')
