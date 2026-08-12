function normalizedDateInput(value) {
	const raw = String(value || '').trim()
	if (!raw) return { raw: '', normalized: '' }
	return {
		raw,
		normalized: raw.includes('T') ? raw : raw.replace(' ', 'T')
	}
}

function defaultInvalidValue(raw, withYear) {
	return raw.replace('T', ' ').slice(withYear ? 0 : 5, 16)
}

export function formatDateTime(value, options = {}) {
	const {
		withYear = true,
		emptyText = '',
		invalidText
	} = options
	const { raw, normalized } = normalizedDateInput(value)
	if (!raw) return emptyText

	const date = new Date(normalized)
	if (Number.isNaN(date.getTime())) {
		if (typeof invalidText === 'function') return invalidText(raw)
		return invalidText === undefined ? defaultInvalidValue(raw, withYear) : invalidText
	}

	const pad = (part) => String(part).padStart(2, '0')
	const dateParts = [pad(date.getMonth() + 1), pad(date.getDate())]
	if (withYear) dateParts.unshift(String(date.getFullYear()))
	return `${dateParts.join('-')} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}
