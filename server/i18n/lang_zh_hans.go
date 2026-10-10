package i18n

// 简体中文：源语言，代码里的提示文案本身就是简体中文，不需要词典，只登记协商用的标签。
func init() {
	Register(Language{Code: Source, Tags: []string{"zh", "zh-cn", "zh-sg", "zh-hans"}}, nil)
}
