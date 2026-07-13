import { resolveAPIURL } from './app-config'
import { callStreamCallback, createDietAssistantStreamParser } from './diet-assistant-sse'
import { createChunkDecoder } from './diet-assistant-stream-decoder'
import { request } from './http'
import { getAccessToken, getSessionState } from './session-storage'

function normalizeMessages(messages = []) {
	return (Array.isArray(messages) ? messages : [])
		.map((message) => ({
			role: String(message?.role || '').trim(),
			content: String(message?.content || '').trim()
		}))
		.filter((message) => message.role && message.content)
}

function getCurrentKitchenId() {
	return Number(getSessionState()?.currentKitchenId) || 0
}

function buildMessagesURL(kitchenId = 0, limit = 0) {
	const params = [`kitchenId=${encodeURIComponent(String(kitchenId))}`]
	if (limit > 0) {
		params.push(`limit=${encodeURIComponent(String(limit))}`)
	}
	return `/caipu-api/diet-assistant/messages?${params.join('&')}`
}

export function listDietAssistantMessages(options = {}) {
	const kitchenId = Number(options?.kitchenId) || getCurrentKitchenId()
	const limit = Number(options?.limit) || 50
	if (!kitchenId) {
		return Promise.resolve([])
	}
	return request({
		url: buildMessagesURL(kitchenId, limit),
		method: 'GET'
	}).then((data) => (Array.isArray(data?.items) ? data.items : []))
}

export function clearDietAssistantMessages(options = {}) {
	const kitchenId = Number(options?.kitchenId) || getCurrentKitchenId()
	if (!kitchenId) {
		return Promise.resolve({ deleted: false })
	}
	return request({
		url: buildMessagesURL(kitchenId),
		method: 'DELETE'
	})
}

export function streamDietAssistantChat(messages = [], callbacks = {}) {
	const token = getAccessToken()
	const kitchenId = getCurrentKitchenId()
	const decoder = createChunkDecoder()
	let receivedChunk = false
	let requestTask = null
	let streamError = null
	let settled = false
	let resolveFinished
	let rejectFinished

	function settle(handler, value) {
		if (settled) return
		settled = true
		if (typeof handler === 'function') {
			handler(value)
		}
	}

	function resolveOnce() {
		settle(resolveFinished, undefined)
	}

	function rejectOnce(error) {
		settle(rejectFinished, error)
	}

	const parser = createDietAssistantStreamParser({
		...callbacks,
		onError: (error) => {
			streamError = error
			try {
				callStreamCallback(callbacks.onError, error)
			} finally {
				rejectOnce(error)
			}
		},
		onDone: () => {
			try {
				callStreamCallback(callbacks.onDone)
			} finally {
				resolveOnce()
			}
		}
	})

	const finished = new Promise((resolve, reject) => {
		resolveFinished = resolve
		rejectFinished = reject
		requestTask = uni.request({
			url: resolveAPIURL('/caipu-api/diet-assistant/chat/stream'),
			method: 'POST',
			data: {
				messages: normalizeMessages(messages),
				kitchenId
			},
			header: {
				Authorization: token ? `Bearer ${token}` : '',
				'Content-Type': 'application/json',
				Accept: 'text/event-stream'
			},
			timeout: 90000,
			enableChunked: true,
			responseType: 'arraybuffer',
			success: (response) => {
				const statusCode = Number(response?.statusCode) || 0
				if (statusCode < 200 || statusCode >= 300) {
					rejectOnce(new Error(`饮食管家请求失败 (${statusCode || 'unknown'})`))
					return
				}

				if (!receivedChunk && response?.data) {
					parser.push(decoder.decode(response.data))
				}
				parser.push(decoder.flush())
				parser.flush()
				if (streamError) {
					rejectOnce(streamError)
					return
				}
				resolveOnce()
			},
			fail: (error) => {
				if (settled) return
				rejectOnce(new Error(error?.errMsg || '饮食管家网络请求失败'))
			}
		})

		if (requestTask && typeof requestTask.onChunkReceived === 'function') {
			requestTask.onChunkReceived((chunk) => {
				receivedChunk = true
				parser.push(decoder.decode(chunk?.data || chunk))
			})
		}
	})

	return {
		finished,
		abort() {
			if (requestTask && typeof requestTask.abort === 'function') {
				requestTask.abort()
			}
		},
		setStreamError(error) {
			streamError = error
		}
	}
}
