<script setup lang="ts">
import { diffTwoObj, setSettings } from '@fantastic-admin/settings'
import { useClipboard } from '@vueuse/core'
import eventBus from '@/utils/eventBus'

defineOptions({
  name: 'AppSetting',
})

const route = useRoute()

const appSettingsStore = useAppSettingsStore()
const settingsDefault = setSettings({})
const appMenuStore = useAppMenuStore()

const isShow = ref(false)

const themeRadius = computed<number[]>({
  get() {
    return [appSettingsStore.settings.theme.radius]
  },
  set(value) {
    appSettingsStore.settings.theme.radius = value[0]
  },
})

watch(() => appSettingsStore.settings.menu.mode, (value) => {
  if (value === 'single') {
    appMenuStore.setActived(0)
  }
  else {
    appMenuStore.setActived(route.fullPath)
  }
})
onMounted(() => {
  eventBus.on('global-app-setting-toggle', () => {
    isShow.value = !isShow.value
  })
})

const { copy, copied, isSupported } = useClipboard({ legacy: true })

function handleCopy() {
  copy(JSON.stringify(diffTwoObj(settingsDefault, appSettingsStore.settings), null, 2))
}
</script>

<template>
  <FaModal v-model="isShow" :title="$t('components.appSetting.title')" :description="$t('components.appSetting.description')" :footer="isSupported" :destroy-on-close="false" class="sm:max-w-4xl" content-class="bg-[var(--g-main-area-bg)] transition-background-color">
    <div
      :class="{
        'columns-1': appSettingsStore.mode === 'mobile',
        'columns-2': appSettingsStore.mode === 'pc',
      }"
    >
      <FaPageMain :title="$t('components.appSetting.theme')" class="m-0 mb-4 break-inside-avoid light:border-none" title-class="font-bold" main-class="space-y-4">
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.colorScheme') }}
          </div>
          <FaButtonGroup>
            <FaButton
              v-for="(item, index) in [
                { icon: 'i-ri:sun-line', value: 'light' },
                { icon: 'i-ri:moon-line', value: 'dark' },
                { icon: 'i-codicon:color-mode', value: '' },
              ]" :key="index" :variant="appSettingsStore.settings.theme.colorScheme === item.value ? 'default' : 'outline'" size="sm" :class="{ 'z-1': appSettingsStore.settings.theme.colorScheme === item.value }" @click="appSettingsStore.settings.theme.colorScheme = (item.value as any)"
            >
              <FaIcon :name="item.icon" />
            </FaButton>
          </FaButtonGroup>
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.radius') }}
          </div>
          <FaSlider v-model="themeRadius" :min="0" :max="1" :step="0.25" class="w-1/2" />
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.colorAmblyopia') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.theme.colorAmblyopia" />
        </div>
      </FaPageMain>
      <FaPageMain v-if="appSettingsStore.mode === 'pc'" :title="$t('components.appSetting.menu')" class="m-0 mb-4 break-inside-avoid light:border-none" title-class="font-bold" main-class="space-y-4">
        <div class="menu-mode">
          <FaTooltip :text="$t('components.appSetting.modeSide')" :delay="500">
            <FaButton variant="outline" class="mode mode-side" :class="{ active: appSettingsStore.settings.menu.mode === 'side' }" @click="appSettingsStore.settings.menu.mode = 'side'">
              <div class="mode-container" />
            </FaButton>
          </FaTooltip>
          <FaTooltip :text="$t('components.appSetting.modeHead')" :delay="500">
            <FaButton variant="outline" class="mode mode-head" :class="{ active: appSettingsStore.settings.menu.mode === 'head' }" @click="appSettingsStore.settings.menu.mode = 'head'">
              <div class="mode-container" />
            </FaButton>
          </FaTooltip>
          <FaTooltip :text="$t('components.appSetting.modeSingle')" :delay="500">
            <FaButton variant="outline" class="mode mode-single" :class="{ active: appSettingsStore.settings.menu.mode === 'single' }" @click="appSettingsStore.settings.menu.mode = 'single'">
              <div class="mode-container" />
            </FaButton>
          </FaTooltip>
        </div>
        <div class="setting-item">
          <div class="label" :class="{ 'op-50': !['single', 'side', 'head'].includes(appSettingsStore.settings.menu.mode) }">
            {{ $t('components.appSetting.mainMenuClickMode') }}
            <FaTooltip :text="$t('components.appSetting.mainMenuClickModeTip')">
              <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
            </FaTooltip>
          </div>
          <FaButtonGroup>
            <FaButton
              v-for="(item, index) in [
                { label: $t('components.appSetting.clickSwitch'), value: 'switch' },
                { label: $t('components.appSetting.clickJump'), value: 'jump' },
                { label: $t('components.appSetting.clickSmart'), value: 'smart' },
              ]" :key="index" :variant="appSettingsStore.settings.menu.mainMenuClickMode === item.value ? 'default' : 'outline'" size="sm" :disabled="!['single', 'side', 'head'].includes(appSettingsStore.settings.menu.mode)" :class="{ 'z-1': appSettingsStore.settings.menu.mainMenuClickMode === item.value }" @click="appSettingsStore.settings.menu.mainMenuClickMode = (item.value as any)"
            >
              {{ item.label }}
            </FaButton>
          </FaButtonGroup>
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.subMenuUniqueExpand') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.menu.subMenuUniqueExpand" />
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.subMenuCollapse') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.menu.subMenuCollapse" />
        </div>
        <div v-if="appSettingsStore.mode === 'pc'" class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.subMenuCollapseButton') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.menu.subMenuCollapseButton" />
        </div>
        <div class="setting-item">
          <div class="label" :class="{ 'op-50': appSettingsStore.settings.menu.mode === 'single' }">
            {{ $t('components.appSetting.hotkeys') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.menu.hotkeys" :disabled="appSettingsStore.settings.menu.mode === 'single'" />
        </div>
      </FaPageMain>
      <FaPageMain :title="$t('components.appSetting.topbar')" class="m-0 mb-4 break-inside-avoid light:border-none" title-class="font-bold" main-class="space-y-4">
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.tabbar') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.topbar.tabbar" />
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.toolbar') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.topbar.toolbar" />
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.mode') }}
          </div>
          <FaButtonGroup>
            <FaButton
              v-for="(item, index) in [
                { label: $t('components.appSetting.modeStatic'), value: 'static' },
                { label: $t('components.appSetting.modeFixed'), value: 'fixed' },
                { label: $t('components.appSetting.modeSticky'), value: 'sticky' },
              ]" :key="index" :variant="appSettingsStore.settings.topbar.mode === item.value ? 'default' : 'outline'" size="sm" :class="{ 'z-1': appSettingsStore.settings.topbar.mode === item.value }" :disabled="!appSettingsStore.settings.topbar.tabbar && !appSettingsStore.settings.topbar.toolbar" @click="appSettingsStore.settings.topbar.mode = (item.value as any)"
            >
              {{ item.label }}
            </FaButton>
          </FaButtonGroup>
        </div>
      </FaPageMain>
      <FaPageMain :title="$t('components.appSetting.tabbar')" class="m-0 mb-4 break-inside-avoid light:border-none" title-class="font-bold" main-class="space-y-4">
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.showIcon') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.tabbar.icon" />
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.hotkeys') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.tabbar.hotkeys" />
        </div>
      </FaPageMain>
      <FaPageMain :title="$t('components.appSetting.toolbar')" class="m-0 mb-4 break-inside-avoid light:border-none" title-class="font-bold" main-class="space-y-4">
        <div v-if="appSettingsStore.mode === 'pc'" class="setting-item">
          <div class="label">
            <FaIcon name="i-ic:twotone-double-arrow" />
            {{ $t('components.appSetting.breadcrumb') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.toolbar.breadcrumb" />
        </div>
        <div class="setting-item">
          <div class="label">
            <FaIcon name="i-ri:search-line" />
            {{ $t('components.appSetting.menuSearch') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.toolbar.menuSearch.enable" />
        </div>
        <div class="ps-8 space-y-4">
          <div class="setting-item">
            <div class="label" :class="{ 'op-50': !appSettingsStore.settings.toolbar.menuSearch.enable }">
              {{ $t('components.appSetting.hotkeys') }}
            </div>
            <FaSwitch v-model="appSettingsStore.settings.toolbar.menuSearch.hotkeys" :disabled="!appSettingsStore.settings.toolbar.menuSearch.enable" />
          </div>
        </div>
        <div v-if="appSettingsStore.mode === 'pc'" class="setting-item">
          <div class="label">
            <FaIcon name="i-ri:fullscreen-line" />
            {{ $t('components.appSetting.fullscreen') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.toolbar.fullscreen" />
        </div>
        <div class="setting-item">
          <div class="label">
            <FaIcon name="i-iconoir:refresh-double" />
            {{ $t('components.appSetting.pageReload') }}
            <FaTooltip :text="$t('components.appSetting.pageReloadTip')">
              <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
            </FaTooltip>
          </div>
          <FaSwitch v-model="appSettingsStore.settings.toolbar.pageReload" />
        </div>
        <div class="setting-item">
          <div class="label">
            <FaIcon name="i-ri:sun-line" />
            {{ $t('components.appSetting.colorTheme') }}
          </div>
          <FaSwitch v-model="appSettingsStore.settings.toolbar.colorScheme" />
        </div>
      </FaPageMain>
      <FaPageMain :title="$t('components.appSetting.page')" class="m-0 mb-4 break-inside-avoid light:border-none" title-class="font-bold" main-class="space-y-4">
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.progress') }}
            <FaTooltip :text="$t('components.appSetting.progressTip')">
              <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
            </FaTooltip>
          </div>
          <FaSwitch v-model="appSettingsStore.settings.page.progress" />
        </div>
      </FaPageMain>
      <FaPageMain :title="$t('components.appSetting.app')" class="m-0 mb-4 break-inside-avoid light:border-none" title-class="font-bold" main-class="space-y-4">
        <div class="p-4 pb-4 pt-14 border rounded-lg relative space-y-4">
          <div class="font-bold px-4 py-2 border-b border-e rounded-rb-lg inset-s-0 inset-t-0 absolute">
            {{ $t('components.appSetting.account') }}
          </div>
          <div class="setting-item">
            <div class="label">
              {{ $t('components.appSetting.auth') }}
            </div>
            <FaSwitch v-model="appSettingsStore.settings.app.account.auth" />
          </div>
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.dynamicTitle') }}
            <FaTooltip :text="$t('components.appSetting.dynamicTitleTip')">
              <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
            </FaTooltip>
          </div>
          <FaSwitch v-model="appSettingsStore.settings.app.dynamicTitle" />
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.rip') }}
            <FaTooltip :text="$t('components.appSetting.ripTip')">
              <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
            </FaTooltip>
          </div>
          <FaSwitch v-model="appSettingsStore.settings.app.rip" />
        </div>
        <div class="setting-item">
          <div class="label">
            {{ $t('components.appSetting.mobileAccess') }}
            <FaTooltip :text="$t('components.appSetting.mobileAccessTip')">
              <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
            </FaTooltip>
          </div>
          <FaSwitch v-model="appSettingsStore.settings.app.mobile" />
        </div>
        <div class="p-4 pb-4 pt-14 border rounded-lg relative space-y-4">
          <div class="font-bold px-4 py-2 border-b border-e rounded-rb-lg inset-s-0 inset-t-0 absolute">
            {{ $t('components.appSetting.home') }}
          </div>
          <div class="setting-item">
            <div class="label">
              {{ $t('common.enabled') }}
              <FaTooltip :text="$t('components.appSetting.homeEnableTip')">
                <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
              </FaTooltip>
            </div>
            <FaSwitch v-model="appSettingsStore.settings.app.home.enable" />
          </div>
          <div class="setting-item">
            <div class="label">
              {{ $t('components.appSetting.homeTitle') }}
              <FaTooltip :text="$t('components.appSetting.homeTitleTip')">
                <FaIcon name="i-ri:question-line" class="text-base text-orange cursor-help" />
              </FaTooltip>
            </div>
            <FaInput v-model="appSettingsStore.settings.app.home.title" />
          </div>
        </div>
        <div class="p-4 pb-4 pt-14 border rounded-lg relative space-y-4">
          <div class="font-bold px-4 py-2 border-b border-e rounded-rb-lg inset-s-0 inset-t-0 absolute">
            {{ $t('components.appSetting.copyright') }}
          </div>
          <div class="setting-item">
            <div class="label">
              {{ $t('common.enabled') }}
            </div>
            <FaSwitch v-model="appSettingsStore.settings.app.copyright.enable" />
          </div>
          <div class="setting-item">
            <div class="label">
              {{ $t('components.appSetting.copyrightDate') }}
            </div>
            <FaInput v-model="appSettingsStore.settings.app.copyright.dates" :disabled="!appSettingsStore.settings.app.copyright.enable" />
          </div>
          <div class="setting-item">
            <div class="label">
              {{ $t('components.appSetting.copyrightCompany') }}
            </div>
            <FaInput v-model="appSettingsStore.settings.app.copyright.company" :disabled="!appSettingsStore.settings.app.copyright.enable" />
          </div>
          <div class="setting-item">
            <div class="label">
              {{ $t('components.appSetting.copyrightWebsite') }}
            </div>
            <FaInput v-model="appSettingsStore.settings.app.copyright.website" :disabled="!appSettingsStore.settings.app.copyright.enable" />
          </div>
        </div>
      </FaPageMain>
    </div>
    <template #footer>
      <div class="w-full">
        <div class="text-sm/6 c-rose mb-2 px-4 py-2 text-center rounded-lg bg-rose/20">
          {{ $t('components.appSetting.copyTipPrefix') }}
          <code class="text-sm font-mono font-semibold px-[0.3rem] py-[0.2rem] rounded bg-muted relative">src/settings.ts</code>
          {{ $t('components.appSetting.copyTipSuffix') }}
        </div>
        <FaButton class="w-full" @click="handleCopy">
          <FaIcon :name="copied ? 'i-tabler:clipboard-check' : 'i-tabler:clipboard'" class="size-5" />
          {{ $t('components.appSetting.copySettings') }}
        </FaButton>
      </div>
    </template>
  </FaModal>
</template>

<style scoped>
.menu-mode {
  --uno: flex items-center justify-center gap-4;

  .mode {
    --uno: relative w-16 h-12;

    &.active {
      --uno: ring-primary ring-2;
    }

    &::before,
    &::after,
    .mode-container {
      --uno: absolute pointer-events-none;
    }

    &::before {
      --uno: content-empty bg-primary;
    }

    &::after {
      --uno: content-empty bg-primary/60;
    }

    .mode-container {
      --uno: bg-primary/20 border-width-1.5 border-dashed border-primary;

      &::before {
        --uno: content-empty absolute w-full h-full;
      }
    }

    &-side {
      &::before {
        --uno: top-2 bottom-2 start-2 w-2 rounded-ss-1 rounded-es-1;
      }

      &::after {
        --uno: top-2 bottom-2 start-4.5 w-3;
      }

      .mode-container {
        --uno: inset-t-2 inset-e-2 inset-b-2 inset-s-8 rounded-se-1 rounded-ee-1;
      }
    }

    &-head {
      &::before {
        --uno: top-2 start-2 end-2 h-2 rounded-ss-1 rounded-se-1;
      }

      &::after {
        --uno: top-4.5 start-2 bottom-2 w-3 rounded-es-1;
      }

      .mode-container {
        --uno: inset-t-4.5 inset-e-2 inset-b-2 inset-s-5.5 rounded-ee-1;
      }
    }

    &-single {
      &::after {
        --uno: top-2 start-2 bottom-2 w-3 rounded-ss-1 rounded-es-1;
      }

      .mode-container {
        --uno: inset-t-2 inset-e-2 inset-b-2 inset-s-5.5 rounded-se-1 rounded-ee-1;
      }
    }
  }
}

.setting-item {
  --uno: flex items-center justify-between gap-4;

  .label {
    --uno: flex items-center flex-shrink-0 gap-2 text-sm;
  }
}
</style>
