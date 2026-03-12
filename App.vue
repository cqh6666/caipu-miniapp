<script>
	import { ensureSession } from './utils/auth'

	export default {
		onLaunch: function() {
			ensureSession().catch(() => {})
		},
		onShow: function() {
			console.log('App Show')
		},
		onHide: function() {
			console.log('App Hide')
		}
	}
</script>

<style lang="scss">
	@import "uview-plus/index.scss";

	/*每个页面公共css */
	page {
		background-color: #f6f7fb;
	}
</style>
