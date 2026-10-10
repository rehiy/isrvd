/** 一种界面语言：每个 messages/<code>.ts 默认导出一个 LanguageModule */
export interface LanguageModule {
    /** 语言代码（BCP 47），同时是 localStorage、Accept-Language 与 ?lang= 的取值 */
    code: string
    /** 切换入口里以该语言自己的名称呈现，不参与翻译 */
    label: string
    /** 登录页等空间紧张处的简称 */
    short: string
    /** 写入 <html lang> 的标签，如 'en-US'；缺省取 code */
    htmlLang?: string
    /** 能匹配到该语言的浏览器语言标签（小写），匹配时逐级去掉末尾子标签 */
    tags: string[]
    /** 中文原文 → 译文；未登记的文案回落中文原文 */
    messages: Record<string, string>
}
