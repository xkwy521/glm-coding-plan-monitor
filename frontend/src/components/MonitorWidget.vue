<script lang="ts" setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { EventsOn, WindowGetPosition, WindowSetPosition } from '../../wailsjs/runtime/runtime'
import { RefreshNow, SetWindowSize, LoadConfig, SaveWindowWidth, GetQuotaData } from '../../wailsjs/go/main/App'

interface QuotaData {
  level: string
  five_hour_used_pct: number
  five_hour_left_pct: number
  next_refresh_time: string
  weekly_used_pct: number
  weekly_left_pct: number
  mcp_current: number
  mcp_total: number
  mcp_left: number
  mcp_usage_pct: number
  last_error: string
}

interface Config {
  api_key: string
  refresh_interval: number
  layout_mode: string
  display_items: string[]
  window_width: number
  glass_mode: boolean
  auto_start: boolean
  window_mode: string
}

// 布局常量
const COMPACT_HEIGHT = 50
const EXPANDED_HEIGHT = 380

// 状态
const quota = ref<QuotaData>({
  level: '-',
  five_hour_used_pct: 0,
  five_hour_left_pct: 100,
  next_refresh_time: '-',
  weekly_used_pct: 0,
  weekly_left_pct: 100,
  mcp_current: 0,
  mcp_total: 0,
  mcp_left: 0,
  mcp_usage_pct: 0,
  last_error: '',
})

const config = ref<Config>({
  api_key: '',
  refresh_interval: 5,
  layout_mode: 'single-line',
  display_items: ['5h_used', 'next_refresh'],
  window_width: 200,
  glass_mode: true,
  auto_start: false,
  window_mode: 'floating',
})

const isHovering = ref(false)
const shouldExpandUp = ref(false)
let hoverTimer: ReturnType<typeof setTimeout> | null = null
let resizeTimer: ReturnType<typeof setTimeout> | null = null
let preHoverX = 0
let preHoverY = 0

const isSingleLine = computed(() => config.value.layout_mode === 'single-line')
const hasError = computed(() => !!quota.value.last_error)
const isGlass = computed(() => config.value.glass_mode)

onMounted(async () => {
  try {
    const [cfg, cached] = await Promise.all([LoadConfig(), GetQuotaData()])
    config.value = cfg
    quota.value = cached
  } catch (e) {
    console.warn('初始化失败', e)
  }

  const w = config.value.window_width || 200
  SetWindowSize(w, COMPACT_HEIGHT)

  EventsOn('update_quota', (data: QuotaData) => {
    quota.value = data
  })

  window.addEventListener('resize', onWindowResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onWindowResize)
})

// 窗口 resize 防抖保存
function onWindowResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    const w = window.innerWidth
    if (w > 100 && w < 800) {
      config.value.window_width = w
      SaveWindowWidth(w)
    }
  }, 500)
}

// 悬停时动态调整窗口高度 + 方向判断
watch(isHovering, async (val) => {
  const w = config.value.window_width || 200
  if (val) {
    const pos = await WindowGetPosition()
    const screenH = window.screen.availHeight
    const spaceBelow = screenH - pos.y - COMPACT_HEIGHT
    shouldExpandUp.value = spaceBelow < (EXPANDED_HEIGHT - COMPACT_HEIGHT)

    if (shouldExpandUp.value) {
      preHoverX = pos.x
      preHoverY = pos.y
      WindowSetPosition(pos.x, pos.y - (EXPANDED_HEIGHT - COMPACT_HEIGHT))
    }
    SetWindowSize(w, EXPANDED_HEIGHT)
  } else {
    SetWindowSize(w, COMPACT_HEIGHT)
    if (shouldExpandUp.value) {
      WindowSetPosition(preHoverX, preHoverY)
    }
    shouldExpandUp.value = false
  }
})

// 鼠标悬停
function onMouseEnter() {
  if (hoverTimer) clearTimeout(hoverTimer)
  isHovering.value = true
}

function onMouseLeave() {
  hoverTimer = setTimeout(() => {
    isHovering.value = false
  }, 500)
}

// 右键菜单
function onContextMenu(e: MouseEvent) {
  e.preventDefault()
  showContextMenu.value = true
  menuPos.value = { x: e.clientX, y: e.clientY }
}

const showContextMenu = ref(false)
const menuPos = ref({ x: 0, y: 0 })

function closeMenu() {
  showContextMenu.value = false
}

function handleRefresh() {
  closeMenu()
  RefreshNow()
}

const emit = defineEmits<{
  'open-settings': []
}>()

function handleSettings() {
  closeMenu()
  emit('open-settings')
}

function handleExit() {
  closeMenu()
  import('../../wailsjs/runtime/runtime').then(rt => rt.Quit())
}

function onDocClick() {
  if (showContextMenu.value) closeMenu()
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
})
onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
})

// 分段进度条：50 格，每格 1px
const BAR_SEGMENTS = 50
function filledSegments(pct: number): number {
  return Math.min(BAR_SEGMENTS, Math.max(0, Math.round((pct / 100) * BAR_SEGMENTS)))
}

// 主界面显示项
const displayItemMap: Record<string, { label: string; value: () => string }> = {
  '5h_used': {
    label: '本轮已用',
    value: () => `${quota.value.five_hour_used_pct.toFixed(1)}%`,
  },
  'next_refresh': {
    label: '刷新',
    value: () => quota.value.next_refresh_time || '-',
  },
  'weekly_used': {
    label: '周已用',
    value: () => `${quota.value.weekly_used_pct.toFixed(1)}%`,
  },
  'mcp_usage': {
    label: 'MCP',
    value: () => `${quota.value.mcp_current}/${quota.value.mcp_total}`,
  },
  'level': {
    label: '等级',
    value: () => quota.value.level || '-',
  },
}

const activeDisplayItems = computed(() => {
  return config.value.display_items
    .filter(key => displayItemMap[key])
    .map(key => ({
      key,
      label: displayItemMap[key].label,
      value: displayItemMap[key].value(),
    }))
})
</script>

<template>
  <div
    class="monitor-root"
    :style="{ flexDirection: shouldExpandUp ? 'column-reverse' : 'column' }"
    @mouseenter="onMouseEnter"
    @mouseleave="onMouseLeave"
    @contextmenu="onContextMenu"
  >
    <!-- 主状态栏 -->
    <div class="main-bar" :class="{ 'main-bar-glass': isGlass }" style="--wails-draggable: drag">
      <div
        class="main-display"
        :class="isSingleLine ? 'flex-row items-center' : 'flex-col'"
      >
        <template v-if="hasError">
          <span class="error-text">⚠ {{ quota.last_error }}</span>
        </template>
        <template v-else>
          <template v-for="(item, idx) in activeDisplayItems" :key="item.key">
            <span v-if="idx > 0 && isSingleLine" class="separator">|</span>
            <div :class="isSingleLine ? 'inline-flex items-center gap-1' : 'flex items-center gap-1 py-0.5'">
              <span class="item-label">{{ item.label }}</span>
              <span class="item-value">{{ item.value }}</span>
            </div>
          </template>
        </template>
      </div>
    </div>

    <!-- Hover 详情层 -->
    <Transition name="slide">
      <div
        v-if="isHovering"
        class="detail-panel"
        :class="{
          'detail-panel-glass': isGlass,
          'detail-panel-up': shouldExpandUp
        }"
        style="--wails-draggable: no-drag"
      >
        <div class="detail-title">GLM 使用量监控</div>

        <div class="detail-section">
          <div class="detail-row">
            <span class="detail-label">套餐等级</span>
            <span class="detail-value">{{ quota.level || '-' }}</span>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-subtitle">本轮额度</div>
          <div class="detail-row">
            <div class="segmented-bar">
              <div v-for="i in BAR_SEGMENTS" :key="i" class="bar-seg" :class="{ 'bar-seg-filled': i <= filledSegments(quota.five_hour_used_pct) }"></div>
            </div>
            <span class="detail-value">{{ quota.five_hour_used_pct.toFixed(1) }}%</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">剩余</span>
            <span class="detail-value">{{ quota.five_hour_left_pct.toFixed(1) }}%</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">下次刷新</span>
            <span class="detail-value">{{ quota.next_refresh_time || '-' }}</span>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-subtitle">每周额度</div>
          <div class="detail-row">
            <div class="segmented-bar">
              <div v-for="i in BAR_SEGMENTS" :key="i" class="bar-seg" :class="{ 'bar-seg-filled': i <= filledSegments(quota.weekly_used_pct) }"></div>
            </div>
            <span class="detail-value">{{ quota.weekly_used_pct.toFixed(1) }}%</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">剩余</span>
            <span class="detail-value">{{ quota.weekly_left_pct.toFixed(1) }}%</span>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-subtitle">MCP 调用 (月度)</div>
          <div class="detail-row">
            <span class="detail-label">已用</span>
            <span class="detail-value">{{ quota.mcp_current }} / {{ quota.mcp_total }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">剩余</span>
            <span class="detail-value">{{ quota.mcp_left }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">使用率</span>
            <span class="detail-value">{{ quota.mcp_usage_pct.toFixed(1) }}%</span>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 右键菜单 -->
    <Transition name="fade">
      <div
        v-if="showContextMenu"
        class="context-menu"
        :class="{ 'context-menu-glass': isGlass }"
        :style="{ left: menuPos.x + 'px', top: menuPos.y + 'px' }"
        style="--wails-draggable: no-drag"
      >
        <div class="menu-item" @click="handleRefresh">立即刷新</div>
        <div class="menu-item" @click="handleSettings">设置</div>
        <div class="menu-divider"></div>
        <div class="menu-item menu-item-danger" @click="handleExit">退出</div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.monitor-root {
  width: 100%;
  height: 100%;
  display: flex;
}

/* === 主状态栏 === */
.main-bar {
  padding: 10px 14px;
  background: rgba(30, 30, 46, 0.92);
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #cdd6f4;
  font-family: "Segoe UI", "SF Pro", -apple-system, sans-serif;
  font-size: 13px;
  cursor: default;
}

.main-bar-glass {
  background: rgba(30, 30, 46, 0.55);
  backdrop-filter: blur(20px) saturate(1.8);
  -webkit-backdrop-filter: blur(20px) saturate(1.8);
  border-color: rgba(255, 255, 255, 0.12);
}

.main-display {
  display: flex;
  gap: 6px;
}

.separator {
  color: rgba(255, 255, 255, 0.2);
  margin: 0 2px;
}

.item-label {
  color: #a6adc8;
  font-size: 11px;
}

.item-value {
  color: #cdd6f4;
  font-weight: 500;
  font-size: 13px;
}

.error-text {
  color: #f38ba8;
  font-size: 12px;
}

/* === 详情面板 === */
.detail-panel {
  margin-top: 6px;
  background: rgba(30, 30, 46, 0.95);
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 12px 14px;
  color: #cdd6f4;
  font-family: "Segoe UI", "SF Pro", -apple-system, sans-serif;
  user-select: text;
}

.detail-panel-up {
  margin-top: 0;
  margin-bottom: 6px;
}

.detail-panel-glass {
  background: rgba(30, 30, 46, 0.5);
  backdrop-filter: blur(24px) saturate(1.8);
  -webkit-backdrop-filter: blur(24px) saturate(1.8);
  border-color: rgba(255, 255, 255, 0.14);
}

.detail-title {
  font-size: 13px;
  font-weight: 600;
  color: #cdd6f4;
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.detail-section {
  margin-bottom: 8px;
}
.detail-section:last-child {
  margin-bottom: 0;
}

.detail-subtitle {
  font-size: 10px;
  font-weight: 600;
  color: #89b4fa;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 3px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1px 0;
  gap: 10px;
}

.detail-label {
  color: #a6adc8;
  font-size: 12px;
  white-space: nowrap;
}

.detail-value {
  color: #cdd6f4;
  font-size: 12px;
  font-weight: 500;
  text-align: right;
}

.segmented-bar {
  display: flex;
  gap: 1px;
  height: 6px;
  flex: 1;
}
.bar-seg {
  width: 1px;
  height: 100%;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 0.5px;
}
.bar-seg-filled {
  background: #a6e3a1;
}

/* === 右键菜单 === */
.context-menu {
  position: fixed;
  background: rgba(40, 40, 60, 0.97);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 4px;
  z-index: 200;
  min-width: 120px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
}

.context-menu-glass {
  background: rgba(40, 40, 60, 0.55);
  backdrop-filter: blur(20px) saturate(1.8);
  -webkit-backdrop-filter: blur(20px) saturate(1.8);
}

.menu-item {
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  color: #cdd6f4;
  transition: background 0.15s;
}
.menu-item:hover {
  background: rgba(137, 180, 250, 0.15);
}
.menu-item-danger:hover {
  background: rgba(243, 139, 168, 0.15);
  color: #f38ba8;
}
.menu-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.08);
  margin: 4px 0;
}

/* === 动画 === */
.slide-enter-active { transition: all 0.2s ease-out; }
.slide-leave-active { transition: all 0.15s ease-in; }
.slide-enter-from { opacity: 0; transform: translateY(-8px); }
.slide-leave-to { opacity: 0; transform: translateY(-4px); }

.fade-enter-active, .fade-leave-active { transition: opacity 0.15s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
