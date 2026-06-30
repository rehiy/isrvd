/**
 * WebAuthn 工具层
 *
 * 封装浏览器 WebAuthn API 的调用细节：
 * - base64url ↔ ArrayBuffer 转换
 * - navigator.credentials.create / get 调用
 * - 凭证数据序列化（供后端 go-webauthn 解析）
 *
 * 后端 go-webauthn 直接从 Request.Body 读取凭证，
 * 因此序列化格式必须与 go-webauthn 期望的 JSON 结构完全一致。
 */

import api from '@/service/api'
import type { PasskeyLoginCredential, PasskeyRegisterCredential } from '@/service/types/account'

/**
 * base64url 字符串 -> ArrayBuffer
 * WebAuthn API 要求 challenge / id 等字段为 BufferSource
 */
function base64urlToBuffer(base64url: string): ArrayBuffer {
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
    const bin = atob(base64)
    const buf = new Uint8Array(bin.length)
    for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i)
    return buf.buffer
}

/**
 * ArrayBuffer -> base64url 字符串
 * 用于将 WebAuthn 凭证数据序列化后发送给后端
 */
function bufferToBase64url(buf: ArrayBuffer): string {
    const bin = String.fromCharCode(...new Uint8Array(buf))
    return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

/** 检查当前环境是否支持 WebAuthn */
export function isWebAuthnSupported(): boolean {
    return !!(window.PublicKeyCredential && navigator.credentials)
}

/**
 * 执行 Passkey 注册流程（begin → 浏览器弹窗 → finish）
 *
 * @param displayName 可选，凭证自定义名称
 * @returns 注册成功时 resolve，失败时 reject（含用户友好的错误信息）
 */
export async function registerPasskey(displayName?: string): Promise<void> {
    if (!isWebAuthnSupported()) {
        throw new Error('当前浏览器或环境不支持 Passkey（需要 HTTPS 且浏览器支持 WebAuthn）')
    }

    // 1. 向后端请求注册参数
    const { payload: beginData } = await api.accountPasskeyRegisterBegin(displayName ? { displayName } : {})
    if (!beginData) {
        throw new Error('无法开始 Passkey 注册')
    }

    // 2. 将 base64url 字段转为 ArrayBuffer（WebAuthn API 要求）
    const publicKey = beginData.options.publicKey
    const excludeCredentials: PublicKeyCredentialDescriptor[] | undefined = publicKey.excludeCredentials?.map((item) => ({
        ...item,
        id: base64urlToBuffer(item.id),
    }))
    const creationOptions: CredentialCreationOptions = {
        publicKey: {
            ...publicKey,
            challenge: base64urlToBuffer(publicKey.challenge),
            user: {
                ...publicKey.user,
                id: base64urlToBuffer(publicKey.user.id),
            },
            excludeCredentials,
        },
    }

    // 3. 调用浏览器 WebAuthn API（此处必须在用户手势中调用，确保扩展能识别）
    const credential = await navigator.credentials.create(creationOptions) as PublicKeyCredential | null
    if (!credential) {
        throw new Error('用户取消了 Passkey 注册')
    }

    // 4. 序列化凭证数据，发送给后端完成注册
    //    go-webauthn 直接从 Request.Body 解析，格式必须符合 WebAuthn 规范
    const response = credential.response as AuthenticatorAttestationResponse
    const credentialJSON: PasskeyRegisterCredential = {
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
            attestationObject: bufferToBase64url(response.attestationObject),
            clientDataJSON: bufferToBase64url(response.clientDataJSON),
        },
    }

    await api.accountPasskeyRegisterFinish(beginData.sessionId, credentialJSON)
}

/**
 * 执行 Passkey 登录流程（begin → 浏览器弹窗 → finish）
 *
 * @param username 可选，指定用户名；为空时使用可发现凭证（Discoverable Login）
 * @returns 登录成功时返回 { token, username }
 */
export async function loginWithPasskey(username?: string): Promise<{ token: string; username: string }> {
    if (!isWebAuthnSupported()) {
        throw new Error('当前浏览器或环境不支持 Passkey（需要 HTTPS 且浏览器支持 WebAuthn）')
    }

    // 1. 向后端请求登录参数
    const { payload: beginData } = await api.accountPasskeyLoginBegin({ username })
    if (!beginData) {
        throw new Error('无法开始 Passkey 登录')
    }

    // 2. 将 base64url 字段转为 ArrayBuffer
    const publicKey = beginData.options.publicKey
    const allowCredentials: PublicKeyCredentialDescriptor[] | undefined = publicKey.allowCredentials?.map((item) => ({
        ...item,
        id: base64urlToBuffer(item.id),
    }))
    const requestOptions: CredentialRequestOptions = {
        publicKey: {
            ...publicKey,
            challenge: base64urlToBuffer(publicKey.challenge),
            allowCredentials,
        },
    }

    // 3. 调用浏览器 WebAuthn API
    const credential = await navigator.credentials.get(requestOptions) as PublicKeyCredential | null
    if (!credential) {
        throw new Error('用户取消了 Passkey 认证')
    }

    // 4. 序列化断言数据，发送给后端完成登录
    const response = credential.response as AuthenticatorAssertionResponse
    const credentialJSON: PasskeyLoginCredential = {
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
            authenticatorData: bufferToBase64url(response.authenticatorData),
            clientDataJSON: bufferToBase64url(response.clientDataJSON),
            signature: bufferToBase64url(response.signature),
            userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
        },
    }

    const { payload: loginResult } = await api.accountPasskeyLoginFinish(beginData.sessionId, credentialJSON)

    if (!loginResult) {
        throw new Error('登录失败')
    }

    return loginResult
}
