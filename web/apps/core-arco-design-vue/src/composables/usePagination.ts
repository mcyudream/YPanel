// F1：列表分页 + 搜索三件套（客户端过滤版，后端分页接口直接复用 page/pageSize）。
import { computed, ref } from 'vue'

export function usePagination<T>(source: () => T[], opts?: { pageSize?: number, keyword?: (item: T) => string }) {
  const pageSize = opts?.pageSize ?? 20
  const keyword = ref('')
  const page = ref(1)

  const filtered = computed(() => {
    const kw = keyword.value.trim().toLowerCase()
    const list = source()
    if (!kw) {
      return list
    }
    return list.filter((item) => {
      const text = opts?.keyword ? opts.keyword(item) : JSON.stringify(item)
      return text.toLowerCase().includes(kw)
    })
  })

  const total = computed(() => filtered.value.length)
  const paged = computed(() => {
    const start = (page.value - 1) * pageSize
    return filtered.value.slice(start, start + pageSize)
  })
  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

  function setPage(p: number) {
    page.value = Math.min(Math.max(1, p), totalPages.value)
  }

  // 搜索词变化时回到第一页
  function onKeywordChange() {
    page.value = 1
  }

  return { keyword, page, paged, total, totalPages, setPage, onKeywordChange, pageSize }
}
