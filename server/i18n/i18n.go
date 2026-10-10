// Package i18n 提供服务端提示文案的中英切换能力。
//
// 中文（ZH）是默认语言，也是代码中的书写语言；英文（EN）由词典反查得到。
// 词典以「英文原文 → 中文译文」登记（见 Register），同一份词典服务两个方向：
// ZH 正向查表（英文 → 中文），EN 反向查表（中文 → 英文）。
//
// 语言协商顺序：?lang= 查询参数 → Accept-Language 请求头。
// 未命中词典时保留原文，保证文案不会因为缺少译文而丢失。
package i18n

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/gin-gonic/gin"
)

// Lang 语言标识
type Lang string

// 支持的语言
const (
	ZH Lang = "zh" // 简体中文，默认语言
	EN Lang = "en" // 英文
)

// Default 未能协商出语言时使用的兜底语言
const Default = ZH

const (
	langQuery  = "lang"            // 语言协商：查询参数
	langHeader = "Accept-Language" // 语言协商：请求头

	sepASCII = ": " // 英文语境的分段连接符
	sepHan   = "："  // 中文语境的分段连接符
)

// 词典与两个方向的索引，Register 时整体重建
var (
	mu      sync.RWMutex
	dict    = map[string]string{}
	toZhIdx *dirIndex // 英文 → 中文
	toEnIdx *dirIndex // 中文 → 英文
)

// Parse 解析语言取值（Accept-Language 形式），无法识别时返回 Default。
//
// 支持权重（q=）与多语言列表："en;q=0.8,zh;q=0.5" → EN。
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
		lang := langFromTag(tag)
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

// Register 登记词典条目（英文原文 → 中文译文），同名英文原文以最后登记为准。
func Register(entries map[string]string) {
	if len(entries) == 0 {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for key, value := range entries {
		if key == "" || value == "" {
			continue
		}
		dict[key] = value
	}
	rebuild()
}

// Translate 按目标语言翻译提示文案，未命中词典时返回原文。
//
// 文案按 ": " 分段逐段翻译，兼容 fmt.Errorf("...: %w") 包装出的错误链；
// 相邻两段中前一段为中文时用全角冒号连接，保持中文排版习惯。
func Translate(lang Lang, message string) string {
	if message == "" {
		return message
	}

	idx := indexFor(lang)
	if idx == nil {
		return message
	}

	// 先按整条匹配（含占位符模板），命中则直接返回，
	// 这样 "不支持的证书来源: %s" 这类带参数的文案无需拆分即可翻译
	if translated := idx.lookup(message); translated != message {
		return translated
	}

	// 未命中再按 ": " 分段逐段翻译，兼容 fmt.Errorf("...: %w") 包装出的错误链
	segments := strings.Split(message, sepASCII)
	changed := false
	for i, segment := range segments {
		if translated := idx.lookup(segment); translated != segment {
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
		if lang, ok := ctx.Value(ctxKey{}).(Lang); ok && (lang == ZH || lang == EN) {
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

// ─── 辅助函数 ───

// dirIndex 单个翻译方向的索引：精确匹配 + 带占位符的模板匹配
type dirIndex struct {
	needHan bool              // 为真时只处理含汉字的片段（中文 → 英文方向）
	exact   map[string]string // 无占位符的条目
	tmpls   []tmplEntry       // 含占位符的条目
}

// tmplEntry 带占位符的词典条目
type tmplEntry struct {
	pattern *regexp.Regexp
	head    string // 首个占位符之前的静态前缀，用于快速排除
	literal int    // 非占位部分的长度，越大越具体
	value   string
}

// newDirIndex 创建方向索引
func newDirIndex(needHan bool) *dirIndex {
	return &dirIndex{needHan: needHan, exact: map[string]string{}}
}

// add 登记一条 from → to 的映射，from 含占位符时编译为模板
func (d *dirIndex) add(from, to string) {
	pattern, head, ok := compilePattern(from)
	if !ok {
		d.exact[from] = to
		return
	}
	literal := len(verbPattern.ReplaceAllString(from, ""))
	d.tmpls = append(d.tmpls, tmplEntry{pattern: pattern, head: head, literal: literal, value: to})
}

// sortTemplates 字面量越长的模板越具体，优先命中，避免通用模板抢先匹配
func (d *dirIndex) sortTemplates() {
	sort.SliceStable(d.tmpls, func(i, j int) bool {
		return d.tmpls[i].literal > d.tmpls[j].literal
	})
}

// lookup 翻译单个片段，未命中时返回原文
func (d *dirIndex) lookup(segment string) string {
	if d == nil || segment == "" {
		return segment
	}
	if d.needHan && !containsHan(segment) {
		return segment
	}
	if value, ok := d.exact[segment]; ok {
		return value
	}
	for _, entry := range d.tmpls {
		if entry.head != "" && !strings.HasPrefix(segment, entry.head) {
			continue
		}
		matched := entry.pattern.FindStringSubmatch(segment)
		if matched == nil {
			continue
		}
		return fillTemplate(entry.value, matched[1:])
	}
	return segment
}

// rebuild 依据词典重建两个方向的索引，调用方持有写锁
func rebuild() {
	keys := make([]string, 0, len(dict))
	for key := range dict {
		keys = append(keys, key)
	}
	sort.Strings(keys) // 保证中文译文重复时反查结果稳定

	toZh, toEn := newDirIndex(false), newDirIndex(true)
	for _, key := range keys {
		toZh.add(key, dict[key])
		toEn.add(dict[key], key)
	}
	toZh.sortTemplates()
	toEn.sortTemplates()
	toZhIdx, toEnIdx = toZh, toEn
}

// indexFor 返回目标语言对应的索引
func indexFor(lang Lang) *dirIndex {
	mu.RLock()
	defer mu.RUnlock()
	if lang == EN {
		return toEnIdx
	}
	return toZhIdx
}

// negotiate 语言协商：?lang= → Accept-Language
func negotiate(c *gin.Context) string {
	if value := c.Query(langQuery); value != "" {
		return value
	}
	return c.GetHeader(langHeader)
}

// langFromTag 取语言主标签对应的语言，空字符串表示不支持
func langFromTag(tag string) Lang {
	primary := strings.ToLower(strings.Split(tag, "-")[0])
	// 语言主标签为 2~3 个字母，其余（如 "english"）视为无效
	if len(primary) < 2 || len(primary) > 3 {
		return ""
	}
	for _, r := range primary {
		if !unicode.IsLetter(r) {
			return ""
		}
	}
	switch primary {
	case "zh":
		return ZH
	case "en":
		return EN
	}
	return ""
}

// verbPattern 匹配支持的占位符：%s / %d / %v / %w / %q / %T / %%
var verbPattern = regexp.MustCompile(`%[sdvwqT%]`)

// compilePattern 将含占位符的原文编译为正则，head 为首个占位符之前的静态前缀
func compilePattern(source string) (pattern *regexp.Regexp, head string, ok bool) {
	matched := verbPattern.FindAllStringIndex(source, -1)
	if len(matched) == 0 {
		return nil, "", false
	}

	head = source[:matched[0][0]]
	var expr strings.Builder
	expr.WriteByte('^')
	pos := 0
	for _, verb := range matched {
		expr.WriteString(regexp.QuoteMeta(source[pos:verb[0]]))
		switch source[verb[0]+1] {
		case 'd':
			expr.WriteString(`(\d+)`)
		case '%':
			expr.WriteString(`%`)
		default:
			expr.WriteString(`(.+?)`)
		}
		pos = verb[1]
	}
	expr.WriteString(regexp.QuoteMeta(source[pos:]))
	expr.WriteByte('$')

	compiled, err := regexp.Compile(expr.String())
	if err != nil {
		return nil, "", false
	}
	return compiled, head, true
}

// fillTemplate 按顺序把捕获到的参数回填到译文模板
func fillTemplate(template string, args []string) string {
	var out strings.Builder
	index := 0
	for i := 0; i < len(template); i++ {
		if template[i] != '%' || i+1 >= len(template) {
			out.WriteByte(template[i])
			continue
		}
		switch template[i+1] {
		case 's', 'v', 'd', 'w', 'q', 'T':
			if index < len(args) {
				out.WriteString(args[index])
				index++
			} else {
				out.WriteString(template[i : i+2])
			}
			i++
		case '%':
			out.WriteByte('%')
			i++
		default:
			out.WriteByte(template[i])
		}
	}
	return out.String()
}

// containsHan 判断文本是否包含汉字
func containsHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// snapshot 返回词典快照（英文原文 → 中文译文）与支持的语言
func snapshot() (map[string]string, []Lang) {
	mu.RLock()
	defer mu.RUnlock()

	entries := make(map[string]string, len(dict))
	for key, value := range dict {
		entries[key] = value
	}
	return entries, []Lang{ZH, EN}
}
