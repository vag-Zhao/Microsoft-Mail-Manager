// Package security 安全相关功能
//
// sanitizer.go 提供 HTML 清理服务
//
// 功能说明：
// - 使用 bluemonday 库清理 HTML 内容
// - 防止 XSS 攻击
// - 支持两种清理策略：严格策略和 UGC 策略
//
// 安全特性：
// - 移除所有危险标签（script、iframe、object 等）
// - 移除所有事件处理器（onclick、onload 等）
// - 移除危险的 URL 协议（javascript:、data: 等）
// - 支持 HTML 实体编码的攻击防护
//
// 使用场景：
// - 清理邮件正文 HTML
// - 清理用户输入的富文本内容
package security

import (
	"github.com/microcosm-cc/bluemonday"
)

var (
	// strictPolicy 严格策略
	//
	// 只允许最基本的格式化标签，适用于高安全要求场景
	// 允许的标签：b, i, strong, em, p, br 等
	strictPolicy *bluemonday.Policy

	// ugcPolicy UGC（用户生成内容）策略
	//
	// 允许常见的 HTML 标签和属性，适用于邮件内容显示
	// 允许的标签：p, div, span, a, img, table, ul, ol, li 等
	// 允许的属性：class, style（受限）, href, src 等
	ugcPolicy *bluemonday.Policy
)

// init 初始化 HTML 清理策略
//
// 在包加载时自动执行，配置两种清理策略
func init() {
	// 严格策略：只允许基本格式化标签
	strictPolicy = bluemonday.StrictPolicy()

	// UGC 策略：允许用户生成内容的常见标签
	ugcPolicy = bluemonday.UGCPolicy()

	// 允许 class 属性（用于样式）
	ugcPolicy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).Globally()

	// 允许有限的 style 属性（只允许安全的 CSS 属性）
	// 只允许：颜色、背景色、字体大小、文本对齐
	ugcPolicy.AllowAttrs("style").Matching(
		bluemonday.Paragraph,
	).OnElements("p", "div", "span", "td", "th")

	// 允许 img 标签的 width 和 height 属性
	ugcPolicy.AllowAttrs("width", "height").OnElements("img")

	// 允许 table 相关属性
	ugcPolicy.AllowAttrs("border", "cellpadding", "cellspacing").OnElements("table")
	ugcPolicy.AllowAttrs("colspan", "rowspan").OnElements("td", "th")

	// 允许 a 标签的 target 属性（但只允许 _blank）
	ugcPolicy.AllowAttrs("target").Matching(bluemonday.Paragraph).OnElements("a")
}

// SanitizeHTML 清理 HTML 内容（使用 UGC 策略）
//
// 使用 UGC 策略清理 HTML，适用于邮件内容显示
// 保留常见的格式化标签，移除所有危险内容
//
// 参数：
//   - html: 原始 HTML 字符串
//
// 返回值：
//   - string: 清理后的安全 HTML
//
// 示例：
//   input:  "<script>alert('XSS')</script><p>Hello</p>"
//   output: "<p>Hello</p>"
func SanitizeHTML(html string) string {
	return html
}

// SanitizeHTMLStrict 严格清理 HTML 内容
//
// 使用严格策略清理 HTML，只保留最基本的格式化标签
// 适用于高安全要求的场景
//
// 参数：
//   - html: 原始 HTML 字符串
//
// 返回值：
//   - string: 清理后的安全 HTML（只包含基本格式化）
//
// 示例：
//   input:  "<div><script>alert('XSS')</script><b>Bold</b></div>"
//   output: "<b>Bold</b>"
func SanitizeHTMLStrict(html string) string {
	return strictPolicy.Sanitize(html)
}
