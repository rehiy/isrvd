package i18n

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// catalog 单个语言的词典：精确匹配 + 带占位符的模板匹配
//
// 词典的 key 是源语言（中文）的提示文案，value 是该语言的译文。
type catalog struct {
	entries map[string]string // 全部条目（含模板），供自检与导出使用
	exact   map[string]string // 无占位符的条目，用于 O(1) 精确匹配
	tmpls   []tmplEntry       // 含占位符的条目
}

// tmplEntry 带占位符的词典条目
type tmplEntry struct {
	pattern *regexp.Regexp
	head    string // 首个占位符之前的静态前缀，用于快速排除
	literal int    // 非占位部分的长度，越大越具体
	value   string
}

func newCatalog() *catalog {
	return &catalog{entries: map[string]string{}, exact: map[string]string{}}
}

// add 登记一条 source → translation 的映射，source 含占位符时编译为模板
func (c *catalog) add(source, translation string) {
	c.entries[source] = translation
	pattern, head, ok := compilePattern(source)
	if !ok {
		c.exact[source] = translation
		return
	}
	// 同一模板重复登记时以后者为准
	for i, entry := range c.tmpls {
		if entry.pattern.String() == pattern.String() {
			c.tmpls[i].value = translation
			return
		}
	}
	literal := len(verbPattern.ReplaceAllString(source, ""))
	c.tmpls = append(c.tmpls, tmplEntry{pattern: pattern, head: head, literal: literal, value: translation})
}

// sortTemplates 字面量越长的模板越具体，优先命中，避免通用模板抢先匹配；
// 长度相同时按模式字典序，保证登记顺序（map 遍历顺序）不影响结果
func (c *catalog) sortTemplates() {
	sort.Slice(c.tmpls, func(i, j int) bool {
		if c.tmpls[i].literal != c.tmpls[j].literal {
			return c.tmpls[i].literal > c.tmpls[j].literal
		}
		return c.tmpls[i].pattern.String() < c.tmpls[j].pattern.String()
	})
}

// lookup 翻译单个片段，未命中时返回原文与 false
func (c *catalog) lookup(segment string) (string, bool) {
	if c == nil || segment == "" || !containsHan(segment) {
		return segment, false
	}
	if value, ok := c.exact[segment]; ok {
		return value, true
	}
	for _, entry := range c.tmpls {
		if entry.head != "" && !strings.HasPrefix(segment, entry.head) {
			continue
		}
		if matched := entry.pattern.FindStringSubmatch(segment); matched != nil {
			return fillTemplate(entry.value, matched[1:]), true
		}
	}
	return segment, false
}

// sources 返回词典里登记的全部原文（含模板），按字典序排列
func (c *catalog) sources() []string {
	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
