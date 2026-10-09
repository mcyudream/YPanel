// 查询历史：localStorage 按实例保存（上限 200 条，参考 DBClient，点击回填）。
export interface QueryHistoryItem { sql: string, at: number }

const MAX = 200

function storageKey(instanceId: number) {
  return `ypanel.dbadmin.history.${instanceId}`
}

export function useQueryHistory(instanceId: () => number) {
  const list = ref<QueryHistoryItem[]>([])

  function load() {
    try {
      const raw = localStorage.getItem(storageKey(instanceId()))
      list.value = raw ? (JSON.parse(raw) as QueryHistoryItem[]) : []
    }
    catch {
      list.value = []
    }
  }

  function persist() {
    try {
      localStorage.setItem(storageKey(instanceId()), JSON.stringify(list.value.slice(0, MAX)))
    }
    catch {
      // 配额满等异常忽略（历史非关键数据）
    }
  }

  function push(sql: string) {
    const text = sql.trim()
    if (!text) {
      return
    }
    // 去重置顶
    list.value = [{ sql: text, at: Date.now() }, ...list.value.filter(i => i.sql !== text)].slice(0, MAX)
    persist()
  }

  function clear() {
    list.value = []
    persist()
  }

  watch(instanceId, load, { immediate: true })

  return { list, push, clear }
}
