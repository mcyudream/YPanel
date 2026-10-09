import type { YwAlertOptions, YwConfirmOptions, YwMenuHandle, YwMenuItem, YwMenuOptions, YwUiAdapter } from '@yudream/yudream-webos-core'
/**
 * 内置轻量 UI 适配器（零依赖默认实现）。
 * menu/confirm/alert 通过 render 挂载 body；message 为右上角 toast 队列。
 */
import { h, render } from 'vue'

let toastHost: HTMLDivElement | null = null

function ensureToastHost(): HTMLDivElement {
  if (!toastHost) {
    toastHost = document.createElement('div')
    toastHost.setAttribute('data-yw-toasts', '')
    document.body.appendChild(toastHost)
  }
  return toastHost
}

function toast(type: string, content: string) {
  const host = ensureToastHost()
  const el = document.createElement('div')
  el.className = `yw-toast yw-toast--${type}`
  el.textContent = content
  host.appendChild(el)
  requestAnimationFrame(() => el.classList.add('is-in'))
  setTimeout(() => {
    el.classList.remove('is-in')
    setTimeout(() => el.remove(), 250)
  }, 2600)
}

function menu(options: YwMenuOptions): YwMenuHandle {
  const host = document.createElement('div')
  host.className = 'yw-menu-host'
  document.body.appendChild(host)

  function onGlobalDown(ev: PointerEvent) {
    if (!host.contains(ev.target as Node)) {
      close()
    }
  }

  function close() {
    window.removeEventListener('pointerdown', onGlobalDown, true)
    render(null, host)
    host.remove()
  }

  function renderItems(items: YwMenuItem[]) {
    return items.filter(item => !item.separator).map(item =>
      h('div', {
        class: [
          'yw-menu-item',
          item.danger && 'yw-menu-item--danger',
          item.disabled && 'yw-menu-item--disabled',
        ],
        onClick: (ev: MouseEvent) => {
          ev.stopPropagation()
          if (item.disabled) {
            return
          }
          item.onSelect?.()
          close()
        },
      }, [
        item.icon ? h('i', { class: [item.icon, 'yw-menu-item-icon'] }) : null,
        h('span', item.label),
        item.children ? h('span', { class: 'yw-menu-caret' }, '▸') : null,
      ]),
    )
  }

  const vnode = h('div', {
    class: 'yw-menu',
    style: { left: '0px', top: '0px', visibility: 'hidden' },
  }, renderItems(options.items))

  render(vnode, host)

  // 视口钳制：贴底向上翻转、贴右向左收、留 8px 边距（Dock 等底部菜单否则溢出被裁）
  const el = host.firstElementChild as HTMLElement
  const r = el.getBoundingClientRect()
  const margin = 8
  let left = Math.min(options.x, window.innerWidth - r.width - margin)
  left = Math.max(margin, left)
  let top = Math.min(options.y, window.innerHeight - r.height - margin)
  top = Math.max(margin, top)
  el.style.left = `${left}px`
  el.style.top = `${top}px`
  el.style.visibility = ''

  window.addEventListener('pointerdown', onGlobalDown, true)
  return { close }
}

function overlayCard(content: () => ReturnType<typeof h>) {
  const host = document.createElement('div')
  host.className = 'yw-overlay-host'
  document.body.appendChild(host)
  const close = () => {
    render(null, host)
    host.remove()
  }
  render(h('div', { class: 'yw-overlay' }, [content()]), host)
  return { close }
}

function confirm(options: YwConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    const { close } = overlayCard(() =>
      h('div', { class: 'yw-dialog' }, [
        h('div', { class: 'yw-dialog-title' }, options.title),
        options.content ? h('div', { class: 'yw-dialog-content' }, options.content) : null,
        h('div', { class: 'yw-dialog-actions' }, [
          h('button', {
            class: 'yw-btn',
            onClick: () => {
              close()
              resolve(false)
            },
          }, options.cancelText ?? '取消'),
          h('button', {
            class: ['yw-btn', options.danger ? 'yw-btn--danger' : 'yw-btn--primary'],
            onClick: () => {
              close()
              resolve(true)
            },
          }, options.okText ?? '确定'),
        ]),
      ]))
  })
}

function alert(options: YwAlertOptions): Promise<void> {
  return new Promise((resolve) => {
    const { close } = overlayCard(() =>
      h('div', { class: 'yw-dialog' }, [
        h('div', { class: 'yw-dialog-title' }, options.title),
        options.content ? h('div', { class: 'yw-dialog-content' }, options.content) : null,
        h('div', { class: 'yw-dialog-actions' }, [
          h('button', {
            class: 'yw-btn yw-btn--primary',
            onClick: () => {
              close()
              resolve()
            },
          }, options.okText ?? '确定'),
        ]),
      ]))
  })
}

export const builtinAdapter: YwUiAdapter = {
  name: 'builtin',
  menu,
  confirm,
  alert,
  message: toast,
}

export default builtinAdapter
