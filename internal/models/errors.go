// Package models 数据模型层
//
// errors.go 错误类型定义
//
// 功能说明：
// - 定义详细的错误类型，用于返回操作结果
// - 提供导入操作的详细结果信息
// - 支持错误追踪和用户友好的错误提示
package models

// ImportResult 导入操作结果
//
// 包含导入操作的详细统计信息和错误列表
// 用于向用户展示哪些账号导入成功，哪些失败以及失败原因
type ImportResult struct {
	Total   int           `json:"total"`   // 总数：尝试导入的账号总数
	Success int           `json:"success"` // 成功数：成功导入的账号数量
	Failed  int           `json:"failed"`  // 失败数：导入失败的账号数量
	Errors  []ImportError `json:"errors"`  // 错误详情：每个失败账号的详细信息
}

// ImportError 导入错误详情
//
// 记录单个账号导入失败的详细信息
// 包含邮箱地址、行号和失败原因，便于用户定位问题
type ImportError struct {
	Email  string `json:"email"`  // 失败的邮箱地址
	Line   int    `json:"line"`   // 在导入文本中的行号（从1开始）
	Reason string `json:"reason"` // 失败原因的详细描述
}
