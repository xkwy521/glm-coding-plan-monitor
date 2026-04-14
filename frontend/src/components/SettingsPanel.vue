<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'
import { LoadConfig, SaveConfig, SetWindowSize, SetAutoStart, IsAutoStart } from '../../wailsjs/go/main/App'

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

const form = ref<Config>({
  api_key: '',
  refresh_interval: 5,
  layout_mode: 'single-line',
  display_items: ['5h_used', 'next_refresh'],
  window_width: 200,
  glass_mode: true,
  auto_start: false,
  window_mode: 'floating',
})

const saving = ref(false)
const saveMsg = ref('')
const showApiKey = ref(false)

const displayOptions = [
  { key: '5h_used', label: '本轮已用比例' },
  { key: 'next_refresh', label: '下次刷新时间' },
  { key: 'weekly_used', label: '每周已用比例' },
  { key: 'mcp_usage', label: 'MCP 调用数' },
  { key: 'level', label: '套餐等级' },
]

const MAX_DISPLAY_ITEMS = 4
const canCheckMore = computed(() => form.value.display_items.length < MAX_DISPLAY_ITEMS)

onMounted(async () => {
  try {
    const cfg = await LoadConfig()
    form.value = { ...cfg }
  } catch (e) {
    console.warn('加载配置失败', e)
  }
  // 从注册表读取实际自启状态
  try {
    form.value.auto_start = await IsAutoStart()
  } catch (_) {}
  SetWindowSize(300, 560)
})

function toggleDisplayItem(key: string) {
  const idx = form.value.display_items.indexOf(key)
  if (idx >= 0) {
    form.value.display_items.splice(idx, 1)
  } else if (canCheckMore.value) {
    form.value.display_items.push(key)
  }
}

function isItemChecked(key: string): boolean {
  return form.value.display_items.includes(key)
}

async function handleSave() {
  saving.value = true
  saveMsg.value = ''
  try {
    // 1. 保存配置文件（含重启轮询刷新数据）
    await SaveConfig(form.value)
    // 2. 同步注册表自启状态
    await SetAutoStart(form.value.auto_start)
    saveMsg.value = '保存成功'
    setTimeout(() => { saveMsg.value = '' }, 2000)
  } catch (e: any) {
    saveMsg.value = '保存失败: ' + (e.message || e)
  } finally {
    saving.value = false
  }
}

const emit = defineEmits<{
  close: []
}>()
</script>

<template>
  <div class="settings-panel" style="--wails-draggable: no-drag">
    <div class="settings-header">
      <span class="settings-title">设置</span>
      <button class="close-btn" @click="emit('close')">×</button>
    </div>

    <div class="settings-body">
      <!-- API Key -->
      <div class="form-group">
        <label class="form-label">API Key</label>
        <div class="password-input">
          <input
            v-model="form.api_key"
            :type="showApiKey ? 'text' : 'password'"
            class="form-input"
            placeholder="输入智谱 API Key"
          />
          <button class="toggle-visibility" @click="showApiKey = !showApiKey">
            {{ showApiKey ? '隐藏' : '显示' }}
          </button>
        </div>
      </div>

      <!-- 刷新频率 -->
      <div class="form-group">
        <label class="form-label">刷新频率</label>
        <select v-model.number="form.refresh_interval" class="form-select">
          <option :value="3">每 3 分钟</option>
          <option :value="5">每 5 分钟</option>
          <option :value="10">每 10 分钟</option>
        </select>
      </div>

      <!-- 排版模式 -->
      <div class="form-group">
        <label class="form-label">排版模式</label>
        <div class="radio-group">
          <label class="radio-label">
            <input type="radio" v-model="form.layout_mode" value="single-line" />
            <span>单行紧凑</span>
          </label>
          <label class="radio-label">
            <input type="radio" v-model="form.layout_mode" value="multi-line" />
            <span>多行堆叠</span>
          </label>
        </div>
      </div>

      <!-- 窗口模式 -->
      <div class="form-group">
        <label class="form-label">窗口模式</label>
        <div class="radio-group">
          <label class="radio-label">
            <input type="radio" v-model="form.window_mode" value="floating" />
            <span>悬浮窗</span>
          </label>
          <label class="radio-label">
            <input type="radio" v-model="form.window_mode" value="tray" />
            <span>托盘图标</span>
          </label>
        </div>
        <div class="form-hint">托盘图标模式下隐藏悬浮窗，仅通过托盘图标查看数据</div>
      </div>

      <!-- 显示参数 -->
      <div class="form-group">
        <label class="form-label">主界面显示项 (最多 {{ MAX_DISPLAY_ITEMS }} 项)</label>
        <div class="checkbox-group">
          <label
            v-for="opt in displayOptions"
            :key="opt.key"
            class="checkbox-label"
            :class="{ disabled: !isItemChecked(opt.key) && !canCheckMore }"
          >
            <input
              type="checkbox"
              :checked="isItemChecked(opt.key)"
              :disabled="!isItemChecked(opt.key) && !canCheckMore"
              @change="toggleDisplayItem(opt.key)"
            />
            <span>{{ opt.label }}</span>
          </label>
        </div>
      </div>

      <!-- 毛玻璃模式 -->
      <div class="form-group">
        <label class="form-label">外观</label>
        <label class="checkbox-label">
          <input type="checkbox" v-model="form.glass_mode" />
          <span>毛玻璃透明效果</span>
        </label>
        <div class="form-hint">关闭后使用不透明深色背景，适合桌面背景干扰时使用</div>
      </div>

      <!-- 开机自启 -->
      <div class="form-group">
        <label class="form-label">系统</label>
        <label class="checkbox-label">
          <input type="checkbox" v-model="form.auto_start" />
          <span>跟随系统启动</span>
        </label>
        <div class="form-hint">勾选后将在 Windows 登录时自动运行</div>
      </div>

      <!-- 提示 -->
      <div class="form-group">
        <div class="form-hint">提示：悬浮窗宽度可直接拖拽窗口边缘调整，会自动保存</div>
      </div>

      <!-- 保存 -->
      <div class="form-actions">
        <button class="save-btn" @click="handleSave" :disabled="saving">
          {{ saving ? '保存中...' : '保存' }}
        </button>
        <span v-if="saveMsg" class="save-msg" :class="{ error: saveMsg.includes('失败') }">
          {{ saveMsg }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-panel {
  width: 100%;
  height: 100%;
  background: rgba(30, 30, 46, 0.95);
  backdrop-filter: blur(16px);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #cdd6f4;
  font-family: "Segoe UI", "SF Pro", -apple-system, sans-serif;
  padding: 16px;
  overflow-y: auto;
}

.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.settings-title {
  font-size: 16px;
  font-weight: 600;
}

.close-btn {
  background: none;
  border: none;
  color: #a6adc8;
  font-size: 20px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}
.close-btn:hover {
  color: #f38ba8;
}

.form-group {
  margin-bottom: 12px;
}

.form-label {
  display: block;
  font-size: 12px;
  color: #a6adc8;
  margin-bottom: 6px;
  font-weight: 500;
}

.form-hint {
  font-size: 11px;
  color: rgba(166, 173, 200, 0.6);
  margin-top: 4px;
}

.form-input {
  width: 100%;
  padding: 8px 10px;
  background: rgba(17, 17, 27, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: #cdd6f4;
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s;
}
.form-input:focus {
  border-color: rgba(137, 180, 250, 0.5);
}

.password-input {
  display: flex;
  gap: 6px;
}
.password-input .form-input {
  flex: 1;
}

.toggle-visibility {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: #a6adc8;
  font-size: 12px;
  cursor: pointer;
  padding: 0 10px;
  white-space: nowrap;
}
.toggle-visibility:hover {
  background: rgba(255, 255, 255, 0.1);
}

.form-select {
  width: 100%;
  padding: 8px 10px;
  background: rgba(17, 17, 27, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: #cdd6f4;
  font-size: 13px;
  outline: none;
  cursor: pointer;
}
.form-select:focus {
  border-color: rgba(137, 180, 250, 0.5);
}

.radio-group {
  display: flex;
  gap: 16px;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  cursor: pointer;
}
.radio-label input[type="radio"] {
  accent-color: #89b4fa;
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  cursor: pointer;
}
.checkbox-label.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.checkbox-label input[type="checkbox"] {
  accent-color: #89b4fa;
}

.form-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 16px;
}

.save-btn {
  padding: 8px 20px;
  background: rgba(137, 180, 250, 0.2);
  border: 1px solid rgba(137, 180, 250, 0.3);
  border-radius: 6px;
  color: #89b4fa;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}
.save-btn:hover {
  background: rgba(137, 180, 250, 0.3);
}
.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.save-msg {
  font-size: 12px;
  color: #a6e3a1;
}
.save-msg.error {
  color: #f38ba8;
}
</style>
