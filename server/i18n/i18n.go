// Package i18n 提供服务端提示文案的多语言能力。
//
// 代码里统一用中文书写提示文案，中文既是默认语言，也是所有语言词典共同的 key：
// 每种语言登记「中文原文 → 译文」，未登记的文案回落中文原文，不会因缺少译文而丢失。
//
// 语言协商顺序：?lang= 查询参数 → Accept-Language 请求头，取值均为 BCP 47 标签。
// 新增语言只需新建 lang_<code>.go 并调用 Register，见 registry.go。
package i18n

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	langQuery  = "lang"            // 语言协商：查询参数
	langHeader = "Accept-Language" // 语言协商：请求头

	sepASCII = ": " // 英文语境的分段连接符
	sepHan   = "："  // 中文语境的分段连接符
)

// Parse 解析语言取值（Accept-Language 形式），无法识别时返回 Default。
//
// 支持权重（q=）与多语言列表："en;q=0.8,zh;q=0.5" → en。
func Parse(spec string) Lang {
	var best Lang
	var bestQ = -1.0

	for _, item := range strings.Split(spec, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		tag, q := item, 1.0
		if i := strings.IndexByte(item, ';'); i >= 0 {
			tag = strings.TrimSpace(item[:i])
			if value, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(item[i+1:]), "q="), 64); err == nil {
				q = value
			}
		}
		lang := resolveTag(tag)
		if lang == "" || q <= bestQ {
			continue
		}
		best, bestQ = lang, q
	}

	if best == "" {
		return Default
	}
	return best
}

// Translate 按目标语言翻译提示文案，未命中词典时返回原文。
//
// 文案按 ": " 分段逐段翻译，兼容 fmt.Errorf("...: %w") 包装出的错误链；
// 相邻两段中前一段为中文时用全角冒号连接，保持中文排版习惯。
func Translate(lang Lang, message string) string {
	if message == "" || lang == Source {
		return message
	}

	cat := catalogFor(lang)
	if cat == nil {
		return message
	}

	// 先按整条匹配（含占位符模板），命中则直接返回，
	// 这样 "不支持的证书来源: %s" 这类带参数的文案无需拆分即可翻译
	if translated, ok := cat.lookup(message); ok {
		return translated
	}

	// 未命中再按 ": " 分段逐段翻译，兼容 fmt.Errorf("...: %w") 包装出的错误链
	segments := strings.Split(message, sepASCII)
	changed := false
	for i, segment := range segments {
		if translated, ok := cat.lookup(segment); ok {
			segments[i] = translated
			changed = true
		}
	}
	if !changed {
		return message
	}

	var out strings.Builder
	for i, segment := range segments {
		if i > 0 {
			if containsHan(segments[i-1]) {
				out.WriteString(sepHan)
			} else {
				out.WriteString(sepASCII)
			}
		}
		out.WriteString(segment)
	}
	return out.String()
}

// Middleware 解析请求语言并写入请求 context，供 T / TC 使用。
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := Parse(negotiate(c))
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxKey{}, lang))
		// 缓存必须按请求头区分（查询参数已体现在 URL 中）
		c.Writer.Header().Add("Vary", langHeader)
		c.Next()
	}
}

// ctxKey 请求 context 中存放语言的键
type ctxKey struct{}

// FromContext 返回 context 携带的语言；未经过中间件时返回 Default。
func FromContext(ctx context.Context) Lang {
	if ctx != nil {
		if lang, ok := ctx.Value(ctxKey{}).(Lang); ok && Known(lang) {
			return lang
		}
	}
	return Default
}

// TC 按 context 携带的语言翻译提示文案，用于拿不到 gin.Context 的服务层。
func TC(ctx context.Context, message string) string {
	return Translate(FromContext(ctx), message)
}

// T 按请求语言翻译提示文案，未命中词典时返回原文。
func T(c *gin.Context, message string) string {
	if c == nil || c.Request == nil {
		return Translate(Default, message)
	}
	return TC(c.Request.Context(), message)
}

// negotiate 语言协商：?lang= → Accept-Language
func negotiate(c *gin.Context) string {
	if value := c.Query(langQuery); value != "" {
		return value
	}
	return c.GetHeader(langHeader)
}
