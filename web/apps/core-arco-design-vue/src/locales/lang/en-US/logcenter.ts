// logcenter domain messages (B26-full): Log Center page (src/views/logcenter/index.vue)
export default {
  title: 'Log Center',
  description: 'Centralized log search (VictoriaLogs / LogsQL): logs from multiple nodes aggregated automatically, with query history, histogram zooming and field-facet filtering',

  status: {
    notConnected: 'Centralized logs not connected',
    recheck: 'Re-check',
    guide1a: 'Install ',
    guide1b: ' from the App Store (one per node that needs aggregated logs; auto-discovered by the panel);',
    guide2a: 'Install ',
    guide2Name: 'Vector log collector',
    guide2b: ' on the same node (default settings suffice; it auto-ships local logs to VL);',
    guide3: 'No address configuration needed — once installed, this page auto-discovers and aggregates logs from all nodes.',
    nodes: 'Aggregating nodes',
    found: 'Discovered',
    notInstalled: 'Not installed',
    offline: 'Offline',
    aggregating: 'Aggregating',
    connected: 'Connected',
  },

  retention: {
    label: 'Retention',
    inputTitle: 'e.g. 7d / 30d / 90d / 1y',
    title: 'Set Retention for All Nodes',
    confirm: 'This will set the VictoriaLogs retention period to {period} on all connected nodes. Containers will restart for a few seconds (queries briefly interrupted); expired data is cleaned up by VL automatically. Continue?',
    applied: 'Applied to {n} nodes',
    partialFailed: 'Some nodes failed',
    setFailed: 'Failed to set',
  },

  fields: {
    label: 'Fields',
    searchPlaceholder: 'Search fields or values',
    follow: 'Follow field',
    unfollow: 'Unfollow',
    noData: '(no data)',
  },

  filters: {
    clear: 'Clear filters ({n})',
    hitToggle: '{n} hits, click to {action} filter',
    addTo: 'add to',
    removeFrom: 'remove from',
  },

  toolbar: {
    simple: 'Simple',
    advanced: 'Advanced',
    historyQuick: 'History & Pinned',
    manage: 'Manage…',
    to: 'to',
    keywordPlaceholder: 'Keyword…',
    search: 'Search',
    matchTitle: 'Word matching uses the inverted index, fastest on large log volumes; regex scans line by line and is slowest',
  },

  match: {
    word: 'Word (fast)',
    phrase: 'Phrase contains',
    regex: 'Regex (slow)',
  },

  time: {
    '15m': 'Last 15 minutes',
    '1h': 'Last 1 hour',
    '6h': 'Last 6 hours',
    '24h': 'Last 24 hours',
    '7d': 'Last 7 days',
    custom: 'Custom range',
  },

  limit: {
    l100: 'Limit 100',
    l500: 'Limit 500',
    l1000: 'Limit 1,000',
    l5000: 'Limit 5,000',
    l20000: 'Limit 20,000',
  },

  results: {
    histTitle: 'Click a bar to zoom into that time range',
    hitsPrefix: 'Hits',
    hitsSuffix: 'entries',
    loaded: 'Loaded {n} entries',
    pageLimit: 'Page limit reached',
    newestFirst: 'Newest first',
    oldestFirst: 'Oldest first',
    querying: 'Querying VictoriaLogs…',
    noResults: 'No results (tip: collectors only gather logs produced after startup)',
    emptyHint: 'Once connected, search historical logs by time range / field facets / keywords',
    copyRaw: 'Copy raw',
    copiedRaw: 'Raw text copied',
    copyJson: 'Copy JSON',
    copiedJson: 'JSON copied',
    loadMore: 'Load earlier (loaded {loaded}/{total})',
  },

  history: {
    title: 'Query History & Pinned Queries',
    empty: 'No records yet (queries are recorded automatically)',
    pinTitle: 'Pin as a frequently used query',
    remarkPlaceholder: 'Name…',
    use: 'Use',
    clearHistory: 'Clear history',
    emptyQuery: '(empty query)',
  },

  search: {
    noInstance: 'No usable VictoriaLogs instance found (install the VictoriaLogs app on a node first)',
    customStartRequired: 'Please select a start time',
    queryFailed: 'Query failed',
    loadFailed: 'Failed to load',
  },
}
