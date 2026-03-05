<!--
  App.vue - 邮箱管家主组件

  功能概述：
  - 双视图切换：邮件视图（查看邮件）和管理视图（批量管理账号）
  - 左侧栏：分组列表、账号列表
  - 中间栏：邮件文件夹、邮件列表
  - 右侧栏：邮件内容详情、附件下载
  - 管理视图：统计卡片、账号表格、批量操作
  - 深色模式支持

  组件结构：
  1. script setup - 响应式状态和业务逻辑
  2. template - 页面布局和UI组件
  3. style - 自定义样式（Toast动画、隐藏滚动条）
-->
<script setup lang="ts">
// ============================================================================
// 依赖导入
// ============================================================================
import { ref, onMounted, onBeforeUnmount, watch, computed, nextTick } from 'vue'  // Vue3 Composition API
import { useAccountStore } from './stores/account'      // 账号状态管理
import { useMailStore } from './stores/mail'            // 邮件状态管理
import { formatDate } from './lib/utils'                // 日期格式化工具
// Lucide图标组件
import { Mail, Folder, Users, Plus, Trash2, Upload, ChevronRight, Paperclip, RefreshCw, Copy, Info } from 'lucide-vue-next'

interface AppInfo {
  programName: string
  version: string
  company: string
  copyright: string
}

// ============================================================================
// Store实例
// ============================================================================
const accountStore = useAccountStore()  // 账号Store：管理账号和分组数据
const mailStore = useMailStore()        // 邮件Store：管理邮件和文件夹数据

// ============================================================================
// 核心响应式状态
// ============================================================================
const currentView = ref<'mail' | 'manage'>('mail')  // 当前视图：mail=邮件视图, manage=管理视图
// 深色模式：从localStorage读取初始值，变化时自动保存
const darkMode = ref(localStorage.getItem('darkMode') === 'true')
watch(darkMode, (val) => localStorage.setItem('darkMode', String(val)))
const activeRowId = ref<number | null>(null)          // 当前激活的表格行ID（用于高亮）
const selectedIds = ref<Set<number>>(new Set())       // 批量选中的账号ID集合

// ============================================================================
// 计算属性 - 派生状态
// ============================================================================

/**
 * 是否全选
 * 检查当前筛选后的所有账号是否都被选中
 */
const allSelected = computed(() => {
  const accounts = accountStore.filteredAccounts
  return accounts.length > 0 && accounts.every(a => selectedIds.value.has(a.id))
})

/**
 * 统计数据
 */
const stats = computed(() => {
  const accounts = accountStore.filteredAccounts
  return {
    total: accounts.length,
    selected: selectedIds.value.size,
    groups: accountStore.groups.length,
  }
})

/**
 * 邮件内容HTML
 * 保留邮件原始HTML，避免影响源显示格式和布局
 */
const emailHtmlContent = computed(() => {
  return mailStore.currentMessage?.body?.content || ''
})

// ============================================================================
// 选择操作函数
// ============================================================================

/**
 * 切换单个账号的选中状态
 * @param id - 账号ID
 */
function toggleSelect(id: number) {
  if (selectedIds.value.has(id)) {
    selectedIds.value.delete(id)
  } else {
    selectedIds.value.add(id)
  }
  // 触发响应式更新
  selectedIds.value = new Set(selectedIds.value)
}

/**
 * 切换全选/取消全选
 */
function toggleSelectAll() {
  const accounts = accountStore.filteredAccounts
  if (allSelected.value) {
    selectedIds.value.clear()  // 已全选则取消
  } else {
    accounts.forEach(a => selectedIds.value.add(a.id))  // 未全选则全选
  }
  selectedIds.value = new Set(selectedIds.value)
}

/**
 * 基于邮箱生成唯一固定密码
 * 使用确定性哈希算法，相同邮箱始终生成相同密码
 * @param email - 邮箱地址
 * @returns 12位密码字符串
 */
function generatePassword(email: string): string {
  const seed = 'ZGS2026' + email  // 加盐
  let hash = 0
  // 计算字符串哈希值
  for (let i = 0; i < seed.length; i++) {
    hash = ((hash << 5) - hash) + seed.charCodeAt(i)
    hash = hash & hash
  }
  // 可用字符集（排除易混淆字符如0/O、1/l/I）
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789'
  let pwd = ''
  // 生成12位密码
  for (let i = 0; i < 12; i++) {
    hash = Math.abs((hash * 9301 + 49297) % 233280)
    pwd += chars[hash % chars.length]
  }
  return pwd
}

// ============================================================================
// UI状态变量
// ============================================================================
const importLoading = ref(false)        // 导入中加载状态
const newGroupName = ref('')            // 新建分组名称输入
const showNewGroup = ref(false)         // 是否显示新建分组输入框
const searchKeyword = ref('')           // 账号搜索关键词
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)  // Toast提示状态
const showAboutModal = ref(false)
const appInfo = ref<AppInfo | null>(null)
const emailIframeRef = ref<HTMLIFrameElement | null>(null)
const emailIframeReady = ref(false)

/**
 * 显示Toast提示
 * @param message - 提示消息
 * @param type - 提示类型：success=成功(绿色), error=错误(红色)
 */
function showToast(message: string, type: 'success' | 'error' = 'success') {
  toast.value = { message, type }
  setTimeout(() => toast.value = null, 2500)  // 2.5秒后自动消失
}

/**
 * 过滤垃圾邮件
 * 过滤掉特定的推广邮件（如OpenAI的促销邮件）
 */
const filteredMessages = computed(() => {
  return mailStore.messages.filter(msg => {
    const from = msg.from?.emailAddress?.address?.toLowerCase() || ''
    const preview = msg.bodyPreview || ''
    // 过滤OpenAI促销邮件
    if (from === 'noreply@tm.openai.com' && preview.includes('获享首月免费优惠')) {
      return false
    }
    return true
  })
})

/**
 * 搜索过滤账号
 * 根据关键词搜索过滤账号列表
 */
const searchedAccounts = computed(() => {
  let accounts = accountStore.filteredAccounts
  // 关键词搜索（匹配邮箱）
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (keyword) {
    accounts = accounts.filter(acc => acc.email.toLowerCase().includes(keyword))
  }
  return accounts
})

// ============================================================================
// 右键菜单和确认弹窗
// ============================================================================
const contextMenu = ref<{ type: 'account' | 'group'; id: number; x: number; y: number; flipX: boolean; flipY: boolean; maxHeight: number } | null>(null)  // 右键菜单状态
const confirmModal = ref<{ message: string; onConfirm: () => void } | null>(null)  // 确认弹窗状态
const showMoveGroupMenu = ref(false)
const submenuOffsetY = ref(0)

/**
 * 显示确认弹窗
 * @param message - 确认消息
 * @param onConfirm - 确认回调函数
 */
function showConfirm(message: string, onConfirm: () => void) {
  confirmModal.value = { message, onConfirm }
}

/** 处理确认弹窗的确认操作 */
function handleConfirm() {
  confirmModal.value?.onConfirm()
  confirmModal.value = null
}

/**
 * 显示右键菜单
 * @param e - 鼠标事件
 * @param type - 菜单类型：account=账号菜单, group=分组菜单
 * @param id - 账号或分组ID
 */
function showContextMenu(e: MouseEvent, type: 'account' | 'group', id: number) {
  e.preventDefault()
  showMoveGroupMenu.value = false
  contextMenu.value = { type, id, x: e.clientX, y: e.clientY, flipX: false, flipY: false, maxHeight: 320 }

  window.addEventListener('resize', handleWindowResize)
  // Adjust position after render
  nextTick(() => adjustMenuPosition())
}

/**
 * 调整右键菜单位置，防止溢出视口
 */
function adjustMenuPosition() {
  if (!contextMenu.value) return

  const menuEl = document.querySelector('.context-menu') as HTMLElement
  if (!menuEl) return

  const menuRect = menuEl.getBoundingClientRect()
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight
  const padding = 8

  let x = contextMenu.value.x
  let y = contextMenu.value.y
  let flipX = false
  let flipY = false

  // Check right edge overflow
  if (x + menuRect.width > viewportWidth - padding) {
    x = x - menuRect.width
    flipX = true
  }

  // Check bottom edge overflow
  if (y + menuRect.height > viewportHeight - padding) {
    y = y - menuRect.height
    flipY = true
  }

  // Ensure within bounds
  x = Math.max(padding, Math.min(x, viewportWidth - menuRect.width - padding))
  y = Math.max(padding, Math.min(y, viewportHeight - menuRect.height - padding))

  const availableHeight = Math.max(120, viewportHeight - y - padding)
  contextMenu.value = { ...contextMenu.value, x, y, flipX, flipY, maxHeight: availableHeight }
}

function handleWindowResize() {
  if (!contextMenu.value) return
  nextTick(() => {
    adjustMenuPosition()
    if (showMoveGroupMenu.value) {
      adjustSubmenuPosition()
    }
  })
}

function toggleMoveGroupMenu() {
  showMoveGroupMenu.value = !showMoveGroupMenu.value
  submenuOffsetY.value = 0
  if (showMoveGroupMenu.value) {
    nextTick(() => adjustSubmenuPosition())
  }
}

function adjustSubmenuPosition() {
  const submenuEl = document.querySelector('.submenu') as HTMLElement
  if (!submenuEl) return

  const viewportHeight = window.innerHeight
  const padding = 8
  const rect = submenuEl.getBoundingClientRect()

  let offset = 0
  if (rect.bottom > viewportHeight - padding) {
    offset -= (rect.bottom - (viewportHeight - padding))
  }
  if (rect.top + offset < padding) {
    offset += (padding - (rect.top + offset))
  }

  submenuOffsetY.value = offset
}


/** 隐藏右键菜单 */
function hideContextMenu() {
  showMoveGroupMenu.value = false
  submenuOffsetY.value = 0
  contextMenu.value = null
  window.removeEventListener('resize', handleWindowResize)
}

// ============================================================================
// 账号和分组操作函数
// ============================================================================

/**
 * 移动账号到指定分组
 * @param accountId - 账号ID
 * @param groupId - 目标分组ID
 */
async function moveToGroup(accountId: number, groupId: number) {
  console.log(`[App.vue] moveToGroup: accountId=${accountId}, groupId=${groupId}`)
  await accountStore.moveToGroup(accountId, groupId)
  hideContextMenu()
}

/**
 * 刷新账号邮件（清除缓存并重新加载）
 * @param accountId - 账号ID
 */
async function refreshAccount(accountId: number) {
  console.log(`[App.vue] refreshAccount: accountId=${accountId}`)
  hideContextMenu()
  mailStore.clearAccountCache(accountId)
  if (accountStore.selectedAccountId === accountId) {
    await mailStore.loadFolders(accountId, true)
    await mailStore.loadMessages(accountId, mailStore.selectedFolderId || 'inbox', 0, true)
  }
  showToast('已刷新', 'success')
}

/**
 * 复制账号邮箱到剪贴板
 * @param accountId - 账号ID
 */
function copyAccountEmail(accountId: number) {
  const acc = accountStore.accounts.find(a => a.id === accountId)
  if (acc) {
    navigator.clipboard.writeText(acc.email)
    showToast('已复制邮箱', 'success')
  }
  hideContextMenu()
}

/**
 * 删除分组
 * 默认分组只能清空不能删除，其他分组可以删除
 * @param id - 分组ID
 */
async function deleteGroup(id: number) {
  console.log(`[App.vue] deleteGroup: id=${id}`)
  const group = accountStore.groups.find(g => g.id === id)
  const isDefault = group?.name === '默认分组'
  const message = isDefault ? '确定清空默认分组中的所有账号？' : '确定删除此分组？'

  showConfirm(message, async () => {
    // 检查当前选中的账号是否在该分组中
    const selectedAcc = accountStore.accounts.find(a => a.id === accountStore.selectedAccountId)
    const needReset = selectedAcc && selectedAcc.groupId === id

    if (isDefault) {
      await accountStore.clearGroup(id)  // 清空默认分组
    } else {
      await accountStore.deleteGroup(id)  // 删除其他分组
    }

    // 如果当前选中的账号被删除，重置状态
    if (needReset) {
      accountStore.selectedAccountId = null
      mailStore.reset()
    }
  })
  hideContextMenu()
}

async function openAboutModal() {
  try {
    // @ts-ignore
    appInfo.value = await window.go.main.App.GetAppInfo()
  } catch {
    appInfo.value = {
      programName: '邮箱管家',
      version: '1.1.0',
      company: 'ZGS',
      copyright: 'Copyright © 2026 ZGS'
    }
  }
  showAboutModal.value = true
}

// ============================================================================
// 生命周期和监听器
// ============================================================================

/**
 * 组件挂载时初始化数据
 * 1. 加载分组列表
 * 2. 默认选中"默认分组"
 * 3. 加载账号列表
 */
onMounted(async () => {
  // 监听协议更新事件，实时更新账号协议标签
  // @ts-ignore
  window.runtime?.EventsOn('protocol-updated', (accountID: number, protocol: string) => {
    const acc = accountStore.accounts.find(a => a.id === accountID)
    if (acc) acc.protocol = protocol
  })

  await accountStore.loadGroups()
  // 默认选中"默认分组"
  const defaultGroup = accountStore.groups.find(g => g.name === '默认分组')
  if (defaultGroup) {
    accountStore.selectedGroupId = defaultGroup.id
  }
  await accountStore.loadAccounts()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleWindowResize)
})

/**
 * 监听选中账号变化
 * 切换账号时重置邮件状态并加载新账号的文件夹
 */
watch(() => accountStore.selectedAccountId, async (id) => {
  if (id) {
    const success = await mailStore.loadFolders(id)
    // 如果请求被中断或账号已切换，不继续加载邮件
    if (!success || accountStore.selectedAccountId !== id) return
    mailStore.selectedFolderId = 'inbox'
    await mailStore.loadMessages(id, 'inbox', 0)
  }
})

function applyIframeScrollbarStyle() {
  const iframe = emailIframeRef.value
  if (!iframe) return

  try {
    const doc = iframe.contentDocument
    if (!doc) {
      emailIframeReady.value = true
      return
    }

    let styleEl = doc.getElementById('mail-hide-scrollbar-style') as HTMLStyleElement | null
    if (!styleEl) {
      styleEl = doc.createElement('style')
      styleEl.id = 'mail-hide-scrollbar-style'
      doc.head?.appendChild(styleEl)
    }

    styleEl.textContent = `
      html, body, * {
        scrollbar-width: none !important;
        -ms-overflow-style: none !important;
      }
      html::-webkit-scrollbar,
      body::-webkit-scrollbar,
      *::-webkit-scrollbar {
        width: 0 !important;
        height: 0 !important;
        display: none !important;
      }
    `
  } catch {
    // ignore cross-document style errors
  } finally {
    emailIframeReady.value = true
  }
}

watch(() => mailStore.currentMessage?.id, async () => {
  emailIframeReady.value = false
  await nextTick()
  applyIframeScrollbarStyle()
})

// ============================================================================
// 导入和邮件操作函数
// ============================================================================

/**
 * 导入账号文件（自动支持 .zgsacc 与 .txt）
 */
async function importAccountFile() {
  importLoading.value = true
  try {
    const result = await (window as any).go.main.App.ImportAccountsFromFile()
    await accountStore.loadAccounts()
    await accountStore.loadGroups()
    showToast(`导入完成：成功 ${result?.success || 0}，失败 ${result?.failed || 0}`, 'success')
  } catch (e: any) {
    showToast('导入失败: ' + e, 'error')
  } finally {
    importLoading.value = false
  }
}

/**
 * 选择邮件文件夹
 * @param folderId - 文件夹ID
 */
async function selectFolder(folderId: string) {
  console.log(`[App.vue] selectFolder: folderId=${folderId}`)
  mailStore.selectedFolderId = folderId
  if (accountStore.selectedAccountId) {
    await mailStore.loadMessages(accountStore.selectedAccountId, folderId, 0)
  }
}

/**
 * 选择邮件查看详情
 * @param messageId - 邮件ID
 */
async function selectMessage(messageId: string) {
  console.log(`[App.vue] selectMessage: messageId=${messageId}`)
  if (accountStore.selectedAccountId) {
    await mailStore.loadMessageDetail(accountStore.selectedAccountId, messageId)
  }
}

/**
 * 加载更多邮件（分页）
 */
async function loadMore() {
  if (accountStore.selectedAccountId && mailStore.selectedFolderId) {
    await mailStore.loadMessages(accountStore.selectedAccountId, mailStore.selectedFolderId, mailStore.currentPage + 1)
  }
}

/**
 * 创建新分组
 */
async function createGroup() {
  if (!newGroupName.value.trim()) return
  console.log(`[App.vue] createGroup: name=${newGroupName.value}`)
  await accountStore.createGroup(newGroupName.value)
  newGroupName.value = ''
  showNewGroup.value = false
}

/**
 * 删除账号
 * @param id - 账号ID
 */
async function deleteAccount(id: number) {
  console.log(`[App.vue] deleteAccount: id=${id}`)
  showConfirm('确定删除此账号？', async () => {
    await accountStore.deleteAccount(id)
  })
}

// ============================================================================
// 批量操作函数
// ============================================================================

/**
 * 批量删除选中的账号
 */
async function batchDelete() {
  if (selectedIds.value.size === 0) return
  console.log(`[App.vue] batchDelete: count=${selectedIds.value.size}`)
  showConfirm(`确定删除选中的 ${selectedIds.value.size} 个账号？`, async () => {
    await (window as any).go.main.App.DeleteAccounts(Array.from(selectedIds.value))
    selectedIds.value.clear()
    await accountStore.loadAccounts()
    await accountStore.loadGroups()
    showToast('删除成功', 'success')
  })
}

/**
 * 批量移动选中账号到指定分组
 * @param groupId - 目标分组ID
 */
async function batchMoveToGroup(groupId: number) {
  if (selectedIds.value.size === 0) return
  console.log(`[App.vue] batchMoveToGroup: count=${selectedIds.value.size}, groupId=${groupId}`)
  await (window as any).go.main.App.MoveAccountsToGroup(Array.from(selectedIds.value), groupId)
  selectedIds.value.clear()
  await accountStore.loadAccounts()
  await accountStore.loadGroups()
  showToast('移动成功', 'success')
}

// ============================================================================
// Token检测功能
// ============================================================================
const checkingTokens = ref(false)  // 是否正在检测Token
const checkProgress = ref({ current: 0, total: 0 })  // 检测进度

/**
 * 批量检测账号Token有效性
 * 遍历当前筛选的所有账号，逐个检测Token是否有效
 */
async function batchCheckTokens() {
  const accounts = accountStore.filteredAccounts
  if (accounts.length === 0) return
  console.log(`[App.vue] batchCheckTokens: count=${accounts.length}`)
  checkingTokens.value = true
  checkProgress.value = { current: 0, total: accounts.length }
  let success = 0, fail = 0
  // 逐个检测账号Token
  for (const acc of accounts) {
    try {
      const ok = await (window as any).go.main.App.CheckAccountToken(acc.id)
      if (ok) success++
      else fail++
    } catch {
      fail++
    }
    checkProgress.value.current++
  }
  // 刷新账号列表以显示最新状态
  await accountStore.loadAccounts()
  checkingTokens.value = false
  showToast(`检测完成：${success}正常，${fail}异常`, success > fail ? 'success' : 'error')
}

/**
 * 复制账号信息（邮箱+生成的密码）
 * @param acc - 账号对象
 */
function copyAccountInfo(acc: any) {
  const pwd = generatePassword(acc.email)
  const text = `账号：${acc.email}\n密码：${pwd}`
  navigator.clipboard.writeText(text)
  showToast('已复制', 'success')
}

/**
 * 通用复制文本函数
 * @param text - 要复制的文本
 * @param tip - 复制成功提示
 * @param accId - 可选，高亮显示的账号行ID
 */
function copyText(text: string, tip: string, accId?: number) {
  navigator.clipboard.writeText(text)
  showToast(tip, 'success')
  if (accId) activeRowId.value = accId
}

/**
 * 导出当前筛选的账号列表
 * 格式：邮箱,密码（每行一个）
 */
async function exportAccounts() {
  console.log(`[App.vue] exportAccounts: count=${accountStore.filteredAccounts.length}`)
  const lines = accountStore.filteredAccounts.map(acc => `${acc.email},${generatePassword(acc.email)}`)
  const text = lines.join('\n')
  const result = await (window as any).go.main.App.ExportAccountsFile(text)
  if (result) {
    showToast(`已导出 ${lines.length} 个账号`, 'success')
  }
}

/**
 * 导出单个账号（完整导入格式）
 * 格式：邮箱----密码----clientId----refreshToken----分组名
 * @param accountId - 账号ID
 */
async function exportSingleAccount(accountId: number) {
  const acc = accountStore.accounts.find(a => a.id === accountId)
  if (!acc) {
    showToast('账号不存在', 'error')
    hideContextMenu()
    return
  }

  const groupName = accountStore.groups.find(g => g.id === acc.groupId)?.name || acc.groupName || '默认分组'
  const line = `${acc.email}----${acc.password || ''}----${acc.clientId}----${acc.refreshToken || ''}----${groupName}`
  const result = await (window as any).go.main.App.ExportAccountsFile(line)
  if (result) {
    showToast('已导出 1 个账号', 'success')
  }
  hideContextMenu()
}

/**
 * 导出指定分组的所有账号（完整信息）
 * 格式：邮箱----密码----clientId----refreshToken----分组名
 * @param groupId - 分组ID
 */
async function exportGroupAccounts(groupId: number) {
  const group = accountStore.groups.find(g => g.id === groupId)
  const groupName = group?.name || '未知分组'
  const groupAccounts = accountStore.accounts.filter(a => a.groupId === groupId)
  const lines = groupAccounts.map(acc => `${acc.email}----${acc.password || ''}----${acc.clientId}----${acc.refreshToken || ''}----${groupName}`)
  const text = lines.join('\n')
  const result = await (window as any).go.main.App.ExportAccountsFile(text)
  if (result) {
    showToast(`已导出 ${lines.length} 个账号`, 'success')
  }
  hideContextMenu()
}

/**
 * 下载邮件附件
 * 将Base64编码的附件内容转换为可下载文件
 * @param att - 附件对象，包含contentBytes/contentType/name
 */
function downloadAttachment(att: any) {
  if (!att.contentBytes) return
  const link = document.createElement('a')
  link.href = 'data:' + att.contentType + ';base64,' + att.contentBytes
  link.download = att.name
  link.click()
}
</script>

<template>
  <div :class="['h-screen flex flex-col', darkMode ? 'dark bg-gray-900' : 'bg-gray-50']">
    <div class="flex-1 flex overflow-hidden">
    <!-- 左侧栏：分组和账号 -->
    <aside :class="['w-52 border-r flex flex-col text-xs', darkMode ? 'bg-gray-800 border-gray-700 text-gray-200' : 'bg-white']">
      <div :class="['p-4 border-b', darkMode ? 'border-gray-700' : '']">
        <div class="flex items-center justify-between">
          <h1 @click="currentView = currentView === 'mail' ? 'manage' : 'mail'"
            class="text-lg font-semibold flex items-center gap-2 cursor-pointer hover:text-blue-500 transition-colors">
            <Mail class="w-5 h-5 text-blue-500" /> 邮箱管家
          </h1>
          <button
            @click="importAccountFile"
            :disabled="importLoading"
            :class="['p-2 rounded-lg', darkMode ? 'hover:bg-gray-700 disabled:opacity-50' : 'hover:bg-gray-100 disabled:opacity-50', currentView !== 'mail' ? 'invisible' : '']"
            title="导入账号文件">
            <Upload class="w-4 h-4" />
          </button>
        </div>
        <div class="text-[10px] text-gray-400 leading-none mt-0.5 tracking-wide">powered by <span class="bg-gradient-to-r from-blue-500 to-purple-500 bg-clip-text text-transparent font-medium">ZGS</span> in 2026</div>
      </div>

      <!-- 分组列表 -->
      <div :class="['p-3', currentView === 'mail' ? 'border-b' : 'flex-1 overflow-auto', darkMode ? 'border-gray-700' : '']">
        <div class="flex items-center justify-between mb-2">
          <span :class="['text-xs font-medium', darkMode ? 'text-gray-400' : 'text-gray-500']">分组</span>
          <button @click="showNewGroup = !showNewGroup" :class="['p-1 rounded', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
            <Plus class="w-3 h-3" />
          </button>
        </div>
        <div v-if="showNewGroup" class="flex gap-1.5 mb-2">
          <input v-model="newGroupName" @keyup.enter="createGroup" placeholder="分组名称"
            :class="['flex-1 min-w-0 px-2 py-1 text-xs border rounded focus:outline-none focus:ring-1 focus:ring-blue-400', darkMode ? 'bg-gray-700 border-gray-600 text-gray-200' : '']" />
          <button @click="createGroup" class="px-2.5 py-1 bg-blue-500 text-white text-xs rounded hover:bg-blue-600 whitespace-nowrap">添加</button>
        </div>
        <div :class="currentView === 'mail' ? 'max-h-[64px] overflow-y-auto' : ''">
        <div v-for="g in accountStore.groups" :key="g.id"
          @click="accountStore.selectedGroupId = accountStore.selectedGroupId === g.id ? null : g.id"
          @contextmenu="showContextMenu($event, 'group', g.id)"
          :class="['flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer text-sm',
            accountStore.selectedGroupId === g.id ? (darkMode ? 'bg-blue-900/40 text-blue-400' : 'bg-blue-50 text-blue-600') : (darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-50')]">
          <Folder class="w-4 h-4" />
          <span class="flex-1">{{ g.name }}</span>
          <span class="text-xs text-gray-400">{{ g.count }}</span>
        </div>
        </div>
      </div>

      <!-- 账号列表 - 仅邮件视图显示 -->
      <div v-if="currentView === 'mail'" class="flex-1 overflow-auto hide-scrollbar p-3">
        <div class="flex items-center justify-between mb-2">
          <span :class="['text-xs font-medium', darkMode ? 'text-gray-400' : 'text-gray-500']">账号 ({{ searchedAccounts.length }})</span>
          <input v-model="searchKeyword" placeholder="搜索"
            :class="['w-16 px-1.5 py-0.5 text-xs border rounded focus:outline-none focus:ring-1 focus:ring-blue-400', darkMode ? 'bg-gray-700 border-gray-600 text-gray-200' : '']" />
        </div>
        <div v-for="acc in searchedAccounts" :key="acc.id"
          @click="accountStore.selectedAccountId = acc.id"
          @contextmenu="showContextMenu($event, 'account', acc.id)"
          :title="acc.email"
          :class="['group flex items-center gap-2 px-2 py-2 rounded cursor-pointer text-sm mb-1',
            accountStore.selectedAccountId === acc.id ? 'bg-blue-500 text-white' : (darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100')]">
          <div class="w-8 h-8 rounded-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center text-white text-xs font-medium">
            {{ acc.email[0].toUpperCase() }}
          </div>
          <div class="flex-1 min-w-0">
            <div class="truncate">{{ acc.email }}</div>
            <div :class="['text-xs', accountStore.selectedAccountId === acc.id ? 'text-blue-100' : 'text-gray-400']">
              {{ acc.protocol === 'imap' ? '📧 IMAP' : '☁️ O2' }}
            </div>
          </div>
          <button @click.stop="deleteAccount(acc.id)"
            :class="['p-1 rounded opacity-0 group-hover:opacity-100', accountStore.selectedAccountId === acc.id ? 'hover:bg-blue-400' : (darkMode ? 'hover:bg-gray-600' : 'hover:bg-gray-200')]">
            <Trash2 class="w-3 h-3" />
          </button>
        </div>
      </div>
    </aside>

    <!-- 中间栏：文件夹和邮件列表 - 仅邮件视图 -->
    <div v-if="currentView === 'mail'" :class="['w-56 border-r flex flex-col text-xs', darkMode ? 'bg-gray-800 border-gray-700 text-gray-200' : 'bg-white']">
      <!-- 文件夹 - 始终显示 -->
      <div :class="['p-3 border-b', darkMode ? 'border-gray-700' : '']">
        <div :class="['text-xs font-medium mb-2 flex items-center gap-1', darkMode ? 'text-gray-400' : 'text-gray-500']">
          <Folder class="w-3 h-3" /> 文件夹
          <RefreshCw v-if="mailStore.loading" class="w-3 h-3 animate-spin ml-auto" />
        </div>
        <div v-if="mailStore.error" :class="['text-xs text-red-500 mb-2 p-2 rounded', darkMode ? 'bg-red-900/30' : 'bg-red-50']">
          {{ mailStore.error }}
        </div>
        <div v-for="f in mailStore.folders" :key="f.id"
          @click="selectFolder(f.id)"
          :class="['flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer text-sm',
            mailStore.selectedFolderId === f.id ? (darkMode ? 'bg-blue-900/40 text-blue-400' : 'bg-blue-50 text-blue-600') : (darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-50'),
            !accountStore.selectedAccountId ? 'opacity-50' : '']">
          <ChevronRight class="w-3 h-3" />
          <span class="flex-1">{{ f.displayName }}</span>
          <span v-if="f.unreadItemCount || f.totalItemCount" class="px-1.5 py-0.5 bg-blue-500 text-white text-xs rounded-full">
            {{ f.totalItemCount || f.unreadItemCount }}
          </span>
        </div>
      </div>

      <!-- 邮件列表 -->
      <div class="flex-1 overflow-auto">
        <template v-if="accountStore.selectedAccountId && mailStore.selectedFolderId">
          <div v-for="msg in filteredMessages" :key="msg.id"
            @click="selectMessage(msg.id)"
            :class="['px-3 py-2 border-b cursor-pointer', darkMode ? 'border-gray-700 bg-gray-800' : 'bg-white',
              !msg.isRead ? (darkMode ? 'bg-blue-900/30' : 'bg-blue-50') : (darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-50'),
              mailStore.currentMessage?.id === msg.id ? 'border-l-2 border-l-blue-500' : '']">
            <div class="flex items-center gap-2 mb-1">
              <span class="font-medium text-sm truncate flex-1">{{ msg.from?.emailAddress?.name || msg.from?.emailAddress?.address || '未知' }}</span>
              <span class="text-xs text-gray-400">{{ formatDate(msg.receivedDateTime) }}</span>
            </div>
            <div class="text-sm truncate">{{ msg.subject || '(无主题)' }}</div>
            <div class="text-xs text-gray-400 truncate mt-0.5 flex items-center gap-1">
              <Paperclip v-if="msg.hasAttachments" class="w-3 h-3" />
              {{ msg.bodyPreview }}
            </div>
          </div>
          <button v-if="mailStore.messages.length >= 20" @click="loadMore"
            class="w-full py-2 text-sm text-blue-500 hover:bg-gray-50">
            加载更多
          </button>
        </template>
        <div v-else-if="!accountStore.selectedAccountId" class="h-full flex items-center justify-center text-gray-400 text-sm">
          <Users class="w-5 h-5 mr-2" /> 选择账号查看邮件
        </div>
        <div v-else-if="!mailStore.selectedFolderId" class="h-full flex items-center justify-center text-gray-400 text-sm">
          选择文件夹查看邮件
        </div>
      </div>
    </div>

    <!-- 右侧：邮件内容 - 仅邮件视图 -->
    <main v-if="currentView === 'mail'" :class="['flex-1 flex flex-col overflow-hidden', darkMode ? 'bg-gray-900 text-gray-200' : 'bg-white']">
      <!-- 邮件详情加载中 -->
      <div v-if="mailStore.detailLoading" class="h-full flex items-center justify-center">
        <div class="flex flex-col items-center gap-3">
          <div class="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
          <span :class="['text-sm', darkMode ? 'text-gray-400' : 'text-gray-500']">加载中...</span>
        </div>
      </div>
      <template v-else-if="mailStore.currentMessage">
        <div :class="['p-4 border-b shrink-0', darkMode ? 'border-gray-700' : '']">
          <div class="flex items-center justify-between mb-3">
            <h2 class="text-xl font-semibold flex-1">{{ mailStore.currentMessage.subject || '(无主题)' }}</h2>
          </div>
          <div class="flex items-center gap-3 text-sm text-gray-600">
            <div class="w-10 h-10 rounded-full bg-gradient-to-br from-green-400 to-green-600 flex items-center justify-center text-white font-medium">
              {{ (mailStore.currentMessage.from?.emailAddress?.name || mailStore.currentMessage.from?.emailAddress?.address || '?')[0].toUpperCase() }}
            </div>
            <div>
              <div class="font-medium">{{ mailStore.currentMessage.from?.emailAddress?.name || '未知' }}</div>
              <div class="text-gray-400">{{ mailStore.currentMessage.from?.emailAddress?.address }}</div>
            </div>
            <div class="ml-auto text-gray-400">{{ formatDate(mailStore.currentMessage.receivedDateTime) }}</div>
          </div>
        </div>

        <!-- 附件 -->
        <div v-if="mailStore.attachments.length" :class="['px-4 py-2 border-b shrink-0', darkMode ? 'bg-gray-800 border-gray-700' : 'bg-gray-50']">
          <div class="text-xs text-gray-500 mb-1">附件 ({{ mailStore.attachments.length }})</div>
          <div class="flex flex-wrap gap-1">
            <button v-for="att in mailStore.attachments" :key="att.id" @click="downloadAttachment(att)"
              :class="['flex items-center gap-1 px-2 py-1 border rounded text-xs', darkMode ? 'bg-gray-700 border-gray-600 hover:bg-gray-600' : 'bg-white hover:bg-gray-50']">
              <Paperclip class="w-3 h-3" />
              {{ att.name }}
            </button>
          </div>
        </div>

        <!-- 邮件正文 -->
        <div class="flex-1 overflow-hidden">
          <iframe v-if="mailStore.currentMessage.body?.contentType?.toLowerCase() === 'html'"
            ref="emailIframeRef"
            @load="applyIframeScrollbarStyle"
            :srcdoc="emailHtmlContent"
            :class="['w-full h-full border-0 hide-scrollbar', emailIframeReady ? '' : 'invisible']"
            sandbox="allow-same-origin"></iframe>
          <pre v-else :class="['whitespace-pre-wrap text-sm p-4 h-full overflow-auto hide-scrollbar', darkMode ? 'text-gray-200' : '']">{{ mailStore.currentMessage.body?.content || mailStore.currentMessage.bodyPreview }}</pre>
        </div>
      </template>
      <div v-else :class="['h-full flex items-center justify-center', darkMode ? 'text-gray-500' : 'text-gray-400']">
        <Mail class="w-8 h-8 mr-2" /> 选择邮件查看内容
      </div>
    </main>

    <!-- 管理视图 -->
    <main v-if="currentView === 'manage'" :class="['flex-1 flex flex-col overflow-hidden', darkMode ? 'bg-gray-900 text-gray-200' : 'bg-white']">
      <!-- 统计卡片 -->
      <div :class="['p-4 border-b shrink-0 grid grid-cols-3 gap-3', darkMode ? 'border-gray-700' : '']">
        <div :class="['rounded-lg p-3 text-center', darkMode ? 'bg-gray-800' : 'bg-gray-50']">
          <div :class="['text-2xl font-bold', darkMode ? 'text-gray-200' : 'text-gray-700']">{{ stats.total }}</div>
          <div class="text-xs text-gray-500">总账号</div>
        </div>
        <div :class="['rounded-lg p-3 text-center', darkMode ? 'bg-blue-900/30' : 'bg-blue-50']">
          <div class="text-2xl font-bold text-blue-500">{{ stats.selected }}</div>
          <div class="text-xs text-gray-500">已选中</div>
        </div>
        <div :class="['rounded-lg p-3 text-center', darkMode ? 'bg-gray-800' : 'bg-gray-100']">
          <div :class="['text-2xl font-bold', darkMode ? 'text-gray-300' : 'text-gray-600']">{{ stats.groups }}</div>
          <div class="text-xs text-gray-500">分组数</div>
        </div>
      </div>
      <div :class="['p-4 border-b shrink-0 flex items-center justify-between', darkMode ? 'border-gray-700' : '']">
        <div class="flex items-center gap-2">
          <h2 class="text-lg font-semibold">账号管理</h2>
          <span v-if="selectedIds.size > 0" class="text-xs text-gray-500">(已选 {{ selectedIds.size }})</span>
        </div>
        <div class="flex items-center gap-2">
          <template v-if="selectedIds.size > 0">
            <div class="relative group">
              <button :class="['px-2 py-1 text-xs rounded border', darkMode ? 'bg-gray-700 text-gray-100 border-gray-500 hover:bg-gray-600' : 'bg-gray-100 text-gray-700 border-gray-300 hover:bg-gray-200']">移动分组</button>
              <div :class="['absolute right-0 top-full pt-1 hidden group-hover:block z-20']">
                <div :class="['border rounded shadow-lg py-1 min-w-[100px]', darkMode ? 'bg-gray-800 border-gray-700' : 'bg-white']">
                  <button v-for="g in accountStore.groups" :key="g.id" @click="batchMoveToGroup(g.id)"
                    :class="['w-full px-3 py-1 text-left text-xs', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">{{ g.name }}</button>
                </div>
              </div>
            </div>
            <button @click="batchDelete" class="px-2 py-1 text-xs bg-red-100 text-red-600 rounded hover:bg-red-200">删除</button>
          </template>
          <button @click="exportAccounts" class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600">导出</button>
          <button @click="batchCheckTokens" :disabled="checkingTokens" class="px-3 py-1 text-xs bg-green-500 text-white rounded hover:bg-green-600 disabled:opacity-50">
            {{ checkingTokens ? '检测中...' : '检测' }}
          </button>
        </div>
      </div>
      <div class="flex-1 overflow-auto hide-scrollbar">
        <table class="w-full text-sm">
          <thead :class="['sticky top-0 z-10', darkMode ? 'bg-gray-800' : 'bg-gray-50']">
            <tr>
              <th class="w-8 px-2 py-2"><input type="checkbox" :checked="allSelected" @change="toggleSelectAll" class="cursor-pointer" /></th>
              <th :class="['text-left px-3 py-2 font-medium', darkMode ? 'text-gray-300' : 'text-gray-600']">邮箱账号</th>
              <th :class="['text-left px-3 py-2 font-medium w-36', darkMode ? 'text-gray-300' : 'text-gray-600']">App密码</th>
              <th :class="['text-center px-3 py-2 font-medium w-16', darkMode ? 'text-gray-300' : 'text-gray-600']">状态</th>
              <th :class="['text-center px-3 py-2 font-medium w-16', darkMode ? 'text-gray-300' : 'text-gray-600']">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="acc in searchedAccounts" :key="acc.id"
              :class="['border-b', darkMode ? 'border-gray-700' : '', activeRowId === acc.id ? (darkMode ? 'bg-gray-700' : 'bg-gray-100') : selectedIds.has(acc.id) ? (darkMode ? 'bg-blue-900/30' : 'bg-blue-50') : (darkMode ? 'hover:bg-gray-800' : 'hover:bg-gray-50')]">
              <td class="px-2 py-2 text-center"><input type="checkbox" :checked="selectedIds.has(acc.id)" @change="toggleSelect(acc.id)" class="cursor-pointer" /></td>
              <td class="px-3 py-2">
                <span @click="copyText(acc.email, '已复制账号', acc.id)" class="cursor-pointer hover:text-blue-500">{{ acc.email }}</span>
              </td>
              <td class="px-3 py-2">
                <span @click="copyText(generatePassword(acc.email), '已复制密码', acc.id)" class="font-mono text-xs cursor-pointer hover:text-blue-500">{{ generatePassword(acc.email) }}</span>
              </td>
              <td class="px-3 py-2 text-center">
                <span :class="['text-xs', acc.status === 'active' ? 'text-green-500' : 'text-red-500']">
                  {{ acc.status === 'active' ? '正常' : '异常' }}
                </span>
              </td>
              <td class="px-3 py-2 text-center">
                <button @click="copyAccountInfo(acc)" :class="['p-1 rounded', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']" title="复制">
                  <Copy class="w-4 h-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </main>
    </div>

    <!-- 底部状态栏 -->
    <footer :class="['h-7 border-t flex items-center justify-between px-4 text-xs shrink-0', darkMode ? 'bg-gray-800 border-gray-700 text-gray-400' : 'bg-white border-gray-200 text-gray-500']">
      <div class="flex items-center gap-4">
        <span>{{ currentView === 'mail' ? '邮件视图' : '管理视图' }}</span>
        <span>{{ accountStore.groups.length }} 个分组</span>
        <span>{{ accountStore.accounts.length }} 个账号</span>
      </div>
      <div class="flex items-center gap-4">
        <!-- 检测进度条 -->
        <div v-if="checkingTokens" class="flex items-center gap-2">
          <span>检测中 {{ checkProgress.current }}/{{ checkProgress.total }}</span>
          <div :class="['w-32 h-1.5 rounded-full overflow-hidden', darkMode ? 'bg-gray-700' : 'bg-gray-100']">
            <div class="h-full bg-gradient-to-r from-blue-500 to-green-500 rounded-full transition-all duration-300"
              :style="{ width: (checkProgress.total ? (checkProgress.current / checkProgress.total * 100) : 0) + '%' }"></div>
          </div>
        </div>
        <span v-if="mailStore.loading" class="text-blue-500 flex items-center gap-1.5">
          <span class="w-3.5 h-3.5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></span>
          加载中...
        </span>
        <button @click="darkMode = !darkMode"
          :class="['flex items-center gap-1.5 px-2 py-0.5 rounded transition-colors', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
          <span v-if="darkMode">☀️ 浅色</span>
          <span v-else>🌙 深色</span>
        </button>
        <button @click="openAboutModal"
          :class="['flex items-center gap-1.5 px-2 py-0.5 rounded transition-colors', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
          <Info class="w-3.5 h-3.5" />
          <span>关于</span>
        </button>
      </div>
    </footer>



    <!-- 关于弹窗 -->
    <Transition name="modal-fade">
      <div v-if="showAboutModal" @click.self="showAboutModal = false" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div :class="['rounded-lg shadow-xl w-[360px] p-5', darkMode ? 'bg-gray-800 text-gray-200' : 'bg-white']">
          <div class="mb-4">
            <h3 class="font-semibold">关于</h3>
          </div>
          <div class="space-y-2 text-sm">
            <div><span class="text-gray-500">程序名：</span>{{ appInfo?.programName || '邮箱管家' }}</div>
            <div><span class="text-gray-500">版本号：</span>{{ appInfo?.version || '1.1.0' }}</div>
            <div><span class="text-gray-500">公司名：</span>{{ appInfo?.company || 'ZGS' }}</div>
            <div><span class="text-gray-500">Copyright：</span>{{ appInfo?.copyright || 'Copyright © 2026 ZGS' }}</div>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 右键菜单 -->
    <Transition name="menu-pop">
      <div v-if="contextMenu" @click="hideContextMenu" class="fixed inset-0 z-50">
        <div :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px', maxHeight: contextMenu.maxHeight + 'px' }"
          :class="['context-menu absolute border rounded shadow-lg py-1 min-w-[120px] overflow-visible', darkMode ? 'bg-gray-800 border-gray-700 text-gray-200' : 'bg-white']" @click.stop>
        <template v-if="contextMenu.type === 'group'">
          <button @click="exportGroupAccounts(contextMenu.id)" :class="['w-full px-3 py-1.5 text-left text-sm', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
            导出分组
          </button>
          <button @click="deleteGroup(contextMenu.id)" :class="['w-full px-3 py-1.5 text-left text-sm text-red-500', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
            删除分组
          </button>
        </template>
        <template v-else>
          <button @click="refreshAccount(contextMenu.id)" :class="['w-full px-3 py-1.5 text-left text-sm', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
            刷新邮件
          </button>
          <button @click="copyAccountEmail(contextMenu.id)" :class="['w-full px-3 py-1.5 text-left text-sm', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
            复制邮箱
          </button>
          <button @click="exportSingleAccount(contextMenu.id)" :class="['w-full px-3 py-1.5 text-left text-sm', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
            导出此邮箱
          </button>
          <div class="relative">
            <button
              @click="toggleMoveGroupMenu"
              :class="['w-full px-3 py-1.5 text-left text-sm flex items-center justify-between', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
              <span>移动到分组</span>
              <ChevronRight :size="14" :class="showMoveGroupMenu ? 'rotate-90' : ''" />
            </button>
            <div
              v-if="showMoveGroupMenu"
              :style="{ transform: `translateY(${submenuOffsetY}px)` }"
              :class="[
                'submenu absolute py-1 min-w-[140px] border rounded shadow-lg max-h-[240px] overflow-y-auto',
                contextMenu.flipX ? 'right-full mr-1' : 'left-full ml-1',
                'top-0',
                darkMode ? 'bg-gray-800 border-gray-700 text-gray-200' : 'bg-white'
              ]">
              <button
                v-for="g in accountStore.groups"
                :key="g.id"
                @click="moveToGroup(contextMenu.id, g.id)"
                :class="['w-full px-3 py-1.5 text-left text-sm whitespace-nowrap', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">
                {{ g.name }}
              </button>
            </div>
          </div>
        </template>
      </div>
    </div>
    </Transition>

    <!-- 确认弹窗 -->
    <Transition name="modal-fade">
      <div v-if="confirmModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div :class="['rounded-lg shadow-xl w-[300px] p-4', darkMode ? 'bg-gray-800 text-gray-200' : 'bg-white']">
          <p class="text-sm mb-4">{{ confirmModal.message }}</p>
          <div class="flex justify-end gap-2">
            <button @click="confirmModal = null" :class="['px-3 py-1.5 text-sm rounded', darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100']">取消</button>
            <button @click="handleConfirm" class="px-3 py-1.5 text-sm bg-red-500 text-white rounded hover:bg-red-600">确定</button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Toast提示 -->
    <Transition name="toast">
      <div v-if="toast" class="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-50">
        <div :class="['px-6 py-3 rounded-lg shadow-lg text-sm', toast.type === 'success' ? 'bg-green-500 text-white' : 'bg-red-500 text-white']">
          {{ toast.message }}
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.toast-enter-from, .toast-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}

.modal-fade-enter-active, .modal-fade-leave-active {
  transition: opacity 0.2s ease;
}
.modal-fade-enter-from, .modal-fade-leave-to {
  opacity: 0;
}
.modal-fade-enter-active > div,
.modal-fade-leave-active > div {
  transition: transform 0.2s ease, opacity 0.2s ease;
}
.modal-fade-enter-from > div,
.modal-fade-leave-to > div {
  transform: translateY(8px) scale(0.98);
  opacity: 0;
}

.menu-pop-enter-active, .menu-pop-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
  transform-origin: top left;
}
.menu-pop-enter-from, .menu-pop-leave-to {
  opacity: 0;
  transform: translateY(4px) scale(0.98);
}

.hide-scrollbar,
.overflow-auto,
.overflow-y-auto,
textarea,
iframe {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.hide-scrollbar::-webkit-scrollbar,
.overflow-auto::-webkit-scrollbar,
.overflow-y-auto::-webkit-scrollbar,
textarea::-webkit-scrollbar,
iframe::-webkit-scrollbar {
  display: none;
  width: 0;
  height: 0;
}

button,
[role='button'] {
  transition: background-color 0.15s ease, color 0.15s ease, border-color 0.15s ease, transform 0.1s ease, opacity 0.15s ease;
}

button:active,
[role='button']:active {
  transform: scale(0.98);
}

button:focus-visible,
input:focus-visible,
select:focus-visible,
textarea:focus-visible {
  outline: none;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.45);
}

.group > button {
  transition: opacity 0.15s ease;
}

tr,
tr td,
.context-menu button,
.submenu button {
  transition: background-color 0.12s ease, color 0.12s ease;
}

/* Context menu submenu styles */
.context-menu {
  overflow: visible;
}

.submenu {
  z-index: 60;
  transition: transform 0.15s ease, opacity 0.15s ease;
}
</style>
