// F11：echarts 按需注册（TreeShakable），业务统一从本模块导入 echarts 与使用。
import * as echarts from 'echarts/core'
import { LineChart, BarChart, PieChart, TreemapChart } from 'echarts/charts'
import {
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
  DataZoomComponent,
  MarkLineComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([
  LineChart,
  BarChart,
  PieChart,
  TreemapChart,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
  DataZoomComponent,
  MarkLineComponent,
  CanvasRenderer,
])

export default echarts
export { echarts }
