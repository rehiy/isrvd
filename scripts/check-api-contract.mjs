#!/usr/bin/env node
/**
 * 校验前端 API 方法和路径是否存在于 OpenAPI 或中控节点网关权限路由中。
 * 参数名及编码表达式按占位路径段比较，Compose/指标网关按类型联合分别展开。
 */
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const read = name => readFileSync(resolve(root, name), 'utf8')
const api = read('webview/src/service/api.ts')
const spec = JSON.parse(read('public/openapi/data.json'))
const nodeRoutes = read('server/service/node/permission.go')

function unionValues(source, typeName) {
    const definition = source.match(new RegExp('^export type ' + typeName + ' = ([^\\n]+)$', 'm'))?.[1]
    if (!definition) {
        throw new Error('找不到路径参数类型 ' + typeName)
    }
    const values = [...definition.matchAll(/'([^']+)'/g)].map(m => m[1])
    if (!values.length) {
        throw new Error('无法提取路径参数 ' + typeName)
    }
    return values
}

const composeTargets = unionValues(read('webview/src/service/types/compose.ts'), 'ComposeDeployTarget')
const gatewayNames = unionValues(read('webview/src/service/types/metrics.ts'), 'MetricsGatewayName')

function canonical(path) {
    return ('/' + path.split('?')[0].replace(/^\/+/, ''))
        .replace(/\$\{[^}]+\}/g, '{param}')
        .replace(/\{[^}]+\}/g, '{param}')
        .replace(/:[A-Za-z0-9_-]+/g, '{param}')
}

function expand(path) {
    if (path.startsWith('compose/' + '$' + '{target}/')) {
        return composeTargets.map(target => path.replace('$' + '{target}', target))
    }
    if (path.startsWith('$' + '{name}/metrics')) {
        return gatewayNames.map(name => path.replace('$' + '{name}', name))
    }
    return [path]
}

const documented = new Set()
for (const [path, operations] of Object.entries(spec.paths)) {
    for (const method of Object.keys(operations)) {
        if (/^(get|post|put|patch|delete)$/.test(method)) {
            documented.add(method + ' ' + canonical(path))
        }
    }
}

// 中控节点接口由独立 net/http 网关实现，不经过 app 的 OpenAPI 生成器。
for (const match of nodeRoutes.matchAll(/\{\s*Route\w+\s*,\s*"(GET|POST|PUT|PATCH|DELETE)"\s*,\s*"([^"]+)"\s*,/g)) {
    documented.add(match[1].toLowerCase() + ' ' + canonical(match[2].replace(/^\/api/, '')))
}

const requestPattern = /\b(?:http|httpBlob)\.(get|post|put|patch|delete)\s*(?:<[^(\n]*>)?\s*\(\s*(['"\x60])([^'"\x60]+)\2/g
const calls = [...api.matchAll(requestPattern)]
if (!calls.length) {
    throw new Error('未识别任何前端 API 调用，请检查扫描规则')
}

let checked = 0
let errors = 0
for (const match of calls) {
    for (const variant of expand(match[3])) {
        checked++
        const route = match[1] + ' ' + canonical(variant)
        if (!documented.has(route)) {
            const line = api.slice(0, match.index).split('\n').length
            console.error('webview/src/service/api.ts:' + line + ' 缺少 API 路由定义: ' + route)
            errors++
        }
    }
}
console.log('已核验 ' + calls.length + ' 处 API 调用、' + checked + ' 个路径变体')
if (errors) {
    console.error('存在 ' + errors + ' 处接口路径/方法不一致')
    process.exit(1)
}
