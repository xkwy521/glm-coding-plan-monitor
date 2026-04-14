<script lang="ts" setup>
import { ref, watch } from 'vue'
import MonitorWidget from './components/MonitorWidget.vue'
import SettingsPanel from './components/SettingsPanel.vue'

const showSettings = ref(false)
</script>

<template>
  <div class="app-root">
    <MonitorWidget
      v-if="!showSettings"
      @open-settings="showSettings = true"
    />
    <SettingsPanel
      v-if="showSettings"
      @close="showSettings = false"
    />
  </div>
</template>

<style scoped>
.app-root {
  width: 100%;
  height: 100%;
  background: transparent;
}
</style>
