package i18n

import (
	"sort"
	"strings"
	"sync"
)

// Lang 语言标识，取 BCP 47 语言代码，如 "zh-Hans"、"zh-Hant"、"en"、"ja"。
type Lang string

// Source 源语言：代码里书写提示文案所用的语言，也是所有词典共同的 key。
// 源语言不需要词典，翻译时原样返回。
const Source Lang = "zh-Hans"

// Default 未能协商出语言时使用的兜底语言
const Default = Source

// Language 一种界面语言的声明
type Language struct {
	// Code 语言代码，也是前端 Accept-Language / ?lang= 携带的取值
	Code Lang
	// Tags 能匹配到该语言的 BCP 47 标签（小写）。匹配时先查完整标签，
	// 再逐级去掉末尾子标签，所以 "zh-hant-tw" 会落到 "zh-hant"，"en-gb" 会落到 "en"。
	Tags []string
}

// 已登记的语言与协商用的标签表，登记发生在各语言文件的 init 里
var (
	mu       sync.RWMutex
	catalogs = map[Lang]*catalog{}
	tagIndex = map[string]Lang{}
)

// Register 登记一种语言及其译文（中文原文 → 译文），可对同一语言多次调用以分文件登记。
//
// 同名原文以最后登记为准；原文含 %s / %d 等占位符时按模板匹配。
// 新增一种语言只需新建 lang_<code>.go，在 init 里调用 Register，不需要改动其他代码。
func Register(language Language, messages map[string]string) {
	mu.Lock()
	defer mu.Unlock()

	cat := catalogs[language.Code]
	if cat == nil {
		cat = newCatalog()
		catalogs[language.Code] = cat
	}
	for _, tag := range language.Tags {
		tagIndex[strings.ToLower(tag)] = language.Code
	}
	for source, translation := range messages {
		if source == "" || translation == "" {
			continue
		}
		cat.add(source, translation)
	}
	cat.sortTemplates()
}

// Languages 返回已登记的语言代码，按代码排序
func Languages() []Lang {
	mu.RLock()
	defer mu.RUnlock()

	langs := make([]Lang, 0, len(catalogs))
	for code := range catalogs {
		langs = append(langs, code)
	}
	sort.Slice(langs, func(i, j int) bool { return langs[i] < langs[j] })
	return langs
}

// Known 判断是否是已登记的语言
func Known(lang Lang) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := catalogs[lang]
	return ok
}

// catalogFor 返回语言的词典，未登记时返回 nil
func catalogFor(lang Lang) *catalog {
	mu.RLock()
	defer mu.RUnlock()
	return catalogs[lang]
}

// resolveTag 把单个 BCP 47 标签解析为已登记的语言，无法识别时返回空串
func resolveTag(tag string) Lang {
	tag = strings.ToLower(strings.TrimSpace(tag))

	mu.RLock()
	defer mu.RUnlock()
	for tag != "" {
		if lang, ok := tagIndex[tag]; ok {
			return lang
		}
		i := strings.LastIndexByte(tag, '-')
		if i < 0 {
			break
		}
		tag = tag[:i]
	}
	return ""
}
